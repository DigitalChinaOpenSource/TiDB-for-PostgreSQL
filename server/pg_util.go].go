package server

import (
	"encoding/binary"
	"github.com/jackc/pgio"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/hack"
	"math"
	"strconv"
	"time"
)

// dumpRowData 向客户端写会RowData
// PgSQL 在扩展查询中，会指定每一列返回数据的格式，可能是Text(0)或者Binary(1)
// 当 resultFormat 只有一个值，代表着整行格式都为Text(0)或者Binary(1)
func dumpRowData(data []byte, columns []*ColumnInfo, row chunk.Row, rf []int16) ([]byte, error) {
	if len(rf) == 1 {
		if rf[0] == 1 {
			return dumpBinaryRowData(data, columns, row)
		}
		return dumpTextRowData(data, columns, row)
	}

	return dumpTextOrBinaryRowData(data, columns, row, rf)
}

// dumpBinaryRowData 向客户端以 Binary 格式写回 RowData
// MySQL 报文协议为小端序，在 PgSQL 中报文为大端序
// 每次只写入一行数据
// 这里只写向缓存，并不发送
func dumpBinaryRowData(data []byte, columns []*ColumnInfo, row chunk.Row) ([]byte, error) {
	data = pgio.AppendUint16(data, uint16(len(columns)))

	for i := range columns {
		if row.IsNull(i) {
			data = pgio.AppendInt32(data, -1)
			continue
		}
		switch columns[i].Type {
		// postgresql does not have a tiny type
		//case mysql.TypeTiny:
		//	data = pgio.AppendInt32(data, 1)
		//	data = append(data, byte(row.GetInt64(i)))
		case mysql.TypeTiny, mysql.TypeShort, mysql.TypeYear:
			data = pgio.AppendInt32(data, 2)
			data = dumpUint16ByBigEndian(data, uint16(row.GetInt64(i)))
		case mysql.TypeInt24, mysql.TypeLong:
			data = pgio.AppendInt32(data, 4)
			data = dumpUint32ByBigEndian(data, uint32(row.GetInt64(i)))
		case mysql.TypeLonglong:
			data = pgio.AppendInt32(data, 8)
			data = dumpUint64ByBigEndian(data, row.GetUint64(i))
		case mysql.TypeFloat:
			data = pgio.AppendInt32(data, 4)
			data = dumpUint32ByBigEndian(data, math.Float32bits(row.GetFloat32(i)))
		case mysql.TypeDouble:
			data = pgio.AppendInt32(data, 8)
			data = dumpUint64ByBigEndian(data, math.Float64bits(row.GetFloat64(i)))
		case mysql.TypeNewDecimal:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetMyDecimal(i).String()))
		case mysql.TypeString, mysql.TypeVarString, mysql.TypeVarchar, mysql.TypeBit,
			mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeBlob:
			data = dumpLengthEncodedStringByBigEndian(data, row.GetBytes(i))
		case mysql.TypeDate, mysql.TypeDatetime, mysql.TypeTimestamp:
			tmp := make([]byte, 0)
			tmp = dumpBinaryDateTimeByBigEndian(tmp, row.GetTime(i))
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeDuration:
			tmp := dumpBinaryTimeByBigEndian(row.GetDuration(i, 0).Duration)
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeEnum:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetEnum(i).String()))
		case mysql.TypeSet:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetSet(i).String()))
		case mysql.TypeJSON:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetJSON(i).String()))
		default:
			return nil, errInvalidType.GenWithStack("invalid type %v", columns[i].Type)
		}
	}

	return data, nil
}

// dumpTextRowData 向客户端以 Text 格式写回 RowData
// 每次只写入一行数据
// 这里只写向缓存，并不发送
func dumpTextRowData(data []byte, columns []*ColumnInfo, row chunk.Row) ([]byte, error) {
	data = pgio.AppendUint16(data, uint16(len(columns)))
	var tmp []byte
	for i, col := range columns {
		if row.IsNull(i) {
			data = pgio.AppendInt32(data, -1)
			continue
		}
		switch col.Type {
		case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong:
			tmp = strconv.AppendInt(nil, row.GetInt64(i), 10)
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeYear:
			year := row.GetInt64(i)
			tmp = nil
			if year == 0 {
				tmp = append(tmp, '0', '0', '0', '0')
			} else {
				tmp = strconv.AppendInt(tmp, year, 10)
			}
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeLonglong:
			if mysql.HasUnsignedFlag(uint(columns[i].Flag)) {
				tmp = strconv.AppendUint(nil, row.GetUint64(i), 10)
			} else {
				tmp = strconv.AppendInt(nil, row.GetInt64(i), 10)
			}
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeFloat:
			prec := -1
			if columns[i].Decimal > 0 && int(col.Decimal) != mysql.NotFixedDec && col.Table == "" {
				prec = int(col.Decimal)
			}
			tmp = appendFormatFloat(nil, float64(row.GetFloat32(i)), prec, 32)
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeDouble:
			prec := types.UnspecifiedLength
			if col.Decimal > 0 && int(col.Decimal) != mysql.NotFixedDec && col.Table == "" {
				prec = int(col.Decimal)
			}
			tmp = appendFormatFloat(nil, row.GetFloat64(i), prec, 64)
			data = dumpLengthEncodedStringByBigEndian(data, tmp)
		case mysql.TypeNewDecimal:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetMyDecimal(i).String()))
		case mysql.TypeString, mysql.TypeVarString, mysql.TypeVarchar, mysql.TypeBit,
			mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeBlob:
			data = dumpLengthEncodedStringByBigEndian(data, row.GetBytes(i))
		case mysql.TypeDate, mysql.TypeDatetime, mysql.TypeTimestamp:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetTime(i).String()))
		case mysql.TypeDuration:
			dur := row.GetDuration(i, int(col.Decimal))
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(dur.String()))
		case mysql.TypeEnum:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetEnum(i).String()))
		case mysql.TypeSet:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetSet(i).String()))
		case mysql.TypeJSON:
			data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetJSON(i).String()))
		default:
			return nil, errInvalidType.GenWithStack("invalid type %v", columns[i].Type)
		}
	}

	return data, nil
}

// dumpTextOrRowData
func dumpTextOrBinaryRowData(data []byte, columns []*ColumnInfo, row chunk.Row, rf []int16) ([]byte, error) {
	data = pgio.AppendUint16(data, uint16(len(columns)))

	for i, col := range columns {
		if row.IsNull(i) {
			data = pgio.AppendInt32(data, -1)
			continue
		}

		tmp := make([]byte, 0, 20)

		// 只有 text 和 binary 两种
		// binary
		if rf[i] == 1 {
			switch col.Type {
			case mysql.TypeTiny, mysql.TypeShort, mysql.TypeYear:
				data = pgio.AppendInt32(data, 2)
				data = dumpUint16ByBigEndian(data, uint16(row.GetInt64(i)))
			case mysql.TypeInt24, mysql.TypeLong:
				data = pgio.AppendInt32(data, 4)
				data = dumpUint32ByBigEndian(data, uint32(row.GetInt64(i)))
			case mysql.TypeLonglong:
				data = pgio.AppendInt32(data, 8)
				data = dumpUint64ByBigEndian(data, row.GetUint64(i))
			case mysql.TypeFloat:
				data = pgio.AppendInt32(data, 4)
				data = dumpUint32ByBigEndian(data, math.Float32bits(row.GetFloat32(i)))
			case mysql.TypeDouble:
				data = pgio.AppendInt32(data, 8)
				data = dumpUint64ByBigEndian(data, math.Float64bits(row.GetFloat64(i)))
			case mysql.TypeNewDecimal:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetMyDecimal(i).String()))
			case mysql.TypeString, mysql.TypeVarString, mysql.TypeVarchar, mysql.TypeBit,
				mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeBlob:
				data = dumpLengthEncodedStringByBigEndian(data, row.GetBytes(i))
			case mysql.TypeDate, mysql.TypeDatetime, mysql.TypeTimestamp:
				tmp = dumpBinaryDateTimeByBigEndian(tmp, row.GetTime(i))
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeDuration:
				tmp = dumpBinaryTimeByBigEndian(row.GetDuration(i, 0).Duration)
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeEnum:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetEnum(i).String()))
			case mysql.TypeSet:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetSet(i).String()))
			case mysql.TypeJSON:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetJSON(i).String()))
			default:
				return nil, errInvalidType.GenWithStack("invalid type %v", col.Type)
			}
		} else {
			switch col.Type {
			case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24, mysql.TypeLong:
				tmp = strconv.AppendInt(nil, row.GetInt64(i), 10)
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeYear:
				year := row.GetInt64(i)
				tmp = nil
				if year == 0 {
					tmp = append(tmp, '0', '0', '0', '0')
				} else {
					tmp = strconv.AppendInt(tmp, year, 10)
				}
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeLonglong:
				if mysql.HasUnsignedFlag(uint(col.Flag)) {
					tmp = strconv.AppendUint(nil, row.GetUint64(i), 10)
				} else {
					tmp = strconv.AppendInt(nil, row.GetInt64(i), 10)
				}
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeFloat:
				prec := -1
				if col.Decimal > 0 && int(col.Decimal) != mysql.NotFixedDec && col.Table == "" {
					prec = int(col.Decimal)
				}
				tmp = appendFormatFloat(nil, float64(row.GetFloat32(i)), prec, 32)
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeDouble:
				prec := types.UnspecifiedLength
				if col.Decimal > 0 && int(col.Decimal) != mysql.NotFixedDec && col.Table == "" {
					prec = int(col.Decimal)
				}
				tmp = appendFormatFloat(nil, row.GetFloat64(i), prec, 64)
				data = dumpLengthEncodedStringByBigEndian(data, tmp)
			case mysql.TypeNewDecimal:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetMyDecimal(i).String()))
			case mysql.TypeString, mysql.TypeVarString, mysql.TypeVarchar, mysql.TypeBit,
				mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeBlob:
				data = dumpLengthEncodedStringByBigEndian(data, row.GetBytes(i))
			case mysql.TypeDate, mysql.TypeDatetime, mysql.TypeTimestamp:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetTime(i).String()))
			case mysql.TypeDuration:
				dur := row.GetDuration(i, int(col.Decimal))
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(dur.String()))
			case mysql.TypeEnum:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetEnum(i).String()))
			case mysql.TypeSet:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetSet(i).String()))
			case mysql.TypeJSON:
				data = dumpLengthEncodedStringByBigEndian(data, hack.Slice(row.GetJSON(i).String()))
			default:
				return nil, errInvalidType.GenWithStack("invalid type %v", col.Type)
			}
		}
	}

	return data, nil
}

func dumpUint16ByBigEndian(buffer []byte, n uint16) []byte {
	buffer = append(buffer, byte(n>>8))
	buffer = append(buffer, byte(n))
	return buffer
}

func dumpUint32ByBigEndian(buffer []byte, n uint32) []byte {
	buffer = append(buffer, byte(n>>24))
	buffer = append(buffer, byte(n>>16))
	buffer = append(buffer, byte(n>>8))
	buffer = append(buffer, byte(n))
	return buffer
}

func dumpUint64ByBigEndian(buffer []byte, n uint64) []byte {
	buffer = append(buffer, byte(n>>56))
	buffer = append(buffer, byte(n>>48))
	buffer = append(buffer, byte(n>>40))
	buffer = append(buffer, byte(n>>32))
	buffer = append(buffer, byte(n>>24))
	buffer = append(buffer, byte(n>>16))
	buffer = append(buffer, byte(n>>8))
	buffer = append(buffer, byte(n))
	return buffer
}

func dumpLengthEncodedStringByBigEndian(buffer []byte, bytes []byte) []byte {
	buffer = pgio.AppendInt32(buffer, int32(len(bytes)))
	buffer = append(buffer, bytes...)
	return buffer
}

// dumpBinaryTimeByBigEndian 将时间转为大端序字节流
func dumpBinaryTimeByBigEndian(dur time.Duration) (data []byte) {
	if dur == 0 {
		return []byte{0}
	}
	data = make([]byte, 13)
	data[0] = 12
	if dur < 0 {
		data[1] = 1
		dur = -dur
	}
	days := dur / (24 * time.Hour)
	dur -= days * 24 * time.Hour
	data[2] = byte(days)
	hours := dur / time.Hour
	dur -= hours * time.Hour
	data[6] = byte(hours)
	minutes := dur / time.Minute
	dur -= minutes * time.Minute
	data[7] = byte(minutes)
	seconds := dur / time.Second
	dur -= seconds * time.Second
	data[8] = byte(seconds)
	if dur == 0 {
		data[0] = 8
		return data[:9]
	}
	binary.BigEndian.PutUint32(data[9:13], uint32(dur/time.Microsecond))
	return
}

func dumpBinaryDateTimeByBigEndian(data []byte, t types.Time) []byte {
	year, mon, day := t.Year(), t.Month(), t.Day()
	switch t.Type() {
	case mysql.TypeTimestamp, mysql.TypeDatetime:
		if t.IsZero() {
			// All zero.
			data = append(data, 0)
		} else if t.Microsecond() != 0 {
			// Has micro seconds.
			data = append(data, 11)
			data = dumpUint16ByBigEndian(data, uint16(year))
			data = append(data, byte(mon), byte(day), byte(t.Hour()), byte(t.Minute()), byte(t.Second()))
			data = dumpUint32ByBigEndian(data, uint32(t.Microsecond()))
		} else if t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
			// Has HH:MM:SS
			data = append(data, 7)
			data = dumpUint16ByBigEndian(data, uint16(year))
			data = append(data, byte(mon), byte(day), byte(t.Hour()), byte(t.Minute()), byte(t.Second()))
		} else {
			// Only YY:MM:DD
			data = append(data, 4)
			data = dumpUint16ByBigEndian(data, uint16(year))
			data = append(data, byte(mon), byte(day))
		}
	case mysql.TypeDate:
		if t.IsZero() {
			data = append(data, 0)
		} else {
			data = append(data, 4)
			data = dumpUint16ByBigEndian(data, uint16(year)) //year
			data = append(data, byte(mon), byte(day))
		}
	}

	return data
}
