package expression

import (
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
)

var(
	_ functionClass	= &currentDatabaseFunctionClass{}
	_ functionClass =&pgSettingsDatabaseFunctionClass{}
)

type currentDatabaseFunctionClass struct {
	baseFunctionClass
}

func (c *currentDatabaseFunctionClass) getFunction(ctx sessionctx.Context, args []Expression)(builtinFunc, error) {
	if err := c.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, c.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinDatabaseSig{bf}
	return sig, nil
}


type pgSettingsDatabaseFunctionClass struct {
	baseFunctionClass
}

func (p *pgSettingsDatabaseFunctionClass) getFunction(ctx sessionctx.Context, args []Expression)(builtinFunc, error) {
	if err := p.verifyArgs(args);err != nil{
		return nil,err
	}
	argTps := make([]types.EvalType, 0, 3)
	argTps = append(argTps, types.ETString, types.ETString)
	if len(args) > 2 {
		argTps = append(argTps, types.ETInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, argTps...)
	if err!=nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgSettingsSig{bf}
	return sig,nil
}

type builtinPgSettingsSig struct {
	baseBuiltinFunc
}

func (b *builtinPgSettingsSig) evalString(row chunk.Row) (string, bool, error) {
	err := b.ctx.GetSessionVars().SetSystemVar(b.args[0].String(),b.args[1].String())
	if err!=nil {
		return "nil" , false , err
	}
	currentSysVals,succ := b.ctx.GetSessionVars().GetSystemVar(b.args[0].String())
	if !succ {
		return "nil", currentSysVals == "",nil
	}
	return currentSysVals, currentSysVals == "", nil
}

type pgEncodingToCharFunctionClass struct {
	baseFunctionClass
}

func (p *pgEncodingToCharFunctionClass) getFunction(ctx sessionctx.Context, args []Expression)(builtinFunc, error){
	if err := p.verifyArgs(args);err != nil{
		return nil,err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName,args, types.ETString, types.ETInt)
	if err!=nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgEncodingToCharSig{bf}
	return sig,nil
}

type builtinPgEncodingToCharSig struct {
	baseBuiltinFunc
}

func (b *builtinPgEncodingToCharSig) Clone() builtinFunc {
	newSig := &builtinPgEncodingToCharSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgEncodingToCharSig) evalString(row chunk.Row)(string, bool, error){
	id, isNull, err := b.args[0].EvalInt(b.ctx,row)
	if isNull || err != nil {
		return "", isNull, err
	}
	charset :=  encodingToChar[id].name
	return charset,  false, nil
}

type EncodingToInt struct{
	name string
	code int
}

var encodingToChar =[]EncodingToInt{
	{"SQL_ASCII", 0},
	{"EUC_JP", 20932},
	{"EUC_CN", 20936},
	{"EUC_KR", 51949},
	{"EUC_TW", 0},
	{"EUC_JIS_2004", 20932},
	{"UTF8", 65001},
	{"MULE_INTERNAL", 0},
	{"LATIN1", 28591},
	{"LATIN2", 28592},
	{"LATIN3", 28593},
	{"LATIN4", 28594},
	{"LATIN5", 28599},
	{"LATIN6", 0},
	{"LATIN7", 0},
	{"LATIN8", 0},
	{"LATIN9", 28605},
	{"LATIN10", 0},
	{"WIN1256", 1256},
	{"WIN1258", 1258},
	{"WIN866", 866},
	{"WIN874", 874},
	{"KOI8R", 20866},
	{"WIN1251", 1251},
	{"WIN1252", 1252},
	{"ISO_8859_5", 28595},
	{"ISO_8859_6", 28596},
	{"ISO_8859_7", 28597},
	{"ISO_8859_8", 28598},
	{"WIN1250", 1250},
	{"WIN1253", 1253},
	{"WIN1254", 1254},
	{"WIN1255", 1255},
	{"WIN1257", 1257},
	{"KOI8U", 21866},
	{"SJIS", 932},
	{"BIG5", 950},
	{"GBK", 936},
	{"UHC", 949},
	{"GB18030", 54936},
	{"JOHAB", 0},
	{"SHIFT_JIS_2004", 932},
}


type pgHasDatabasePrivilegeFunctionClass struct {
	baseFunctionClass
}

func (p *pgHasDatabasePrivilegeFunctionClass) getFunction(ctx sessionctx.Context,args []Expression)(builtinFunc, error){
	if err := p.verifyArgs(args);err != nil{
		return nil,err
	}
	argTps := make([]types.EvalType, 0, 3)
	argTps = append(argTps, types.ETString, types.ETString)
	if len(args) > 2 {
		argTps = append(argTps, types.ETString)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETInt, argTps...)
	if err!=nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgSettingsSig{bf}
	return sig, nil
}

type builtinPgHasDatabasePrivilegeSig struct {
	baseBuiltinFunc
}

func (b *builtinPgHasDatabasePrivilegeSig) Clone() builtinFunc {
	newSig := &builtinPgHasDatabasePrivilegeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgHasDatabasePrivilegeSig) evalString(row chunk.Row)(string, bool, error){
	id, isNull, err := b.args[0].EvalInt(b.ctx,row)
	if isNull || err != nil {
		return "", isNull, err
	}
	charset :=  encodingToChar[id].name
	return charset,  false, nil
}



