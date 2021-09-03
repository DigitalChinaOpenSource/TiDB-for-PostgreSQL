// Copyright 2021 Digital China Group Co.,Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package expression

import (
	"fmt"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
)

var (
	_ functionClass = &currentDatabaseFunctionClass{}
	_ functionClass = &pgSettingsDatabaseFunctionClass{}
)

type currentDatabaseFunctionClass struct {
	baseFunctionClass
}

func (c *currentDatabaseFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
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

//set_config

type pgSettingsDatabaseFunctionClass struct {
	baseFunctionClass
}

func (p *pgSettingsDatabaseFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	argTps := make([]types.EvalType, 0, 3)
	argTps = append(argTps, types.ETString, types.ETString)
	if len(args) > 2 {
		argTps = append(argTps, types.ETInt)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, argTps...)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgSettingsSig{bf}
	return sig, nil
}

type builtinPgSettingsSig struct {
	baseBuiltinFunc
}

func (b *builtinPgSettingsSig) evalString(row chunk.Row) (string, bool, error) {
	err := b.ctx.GetSessionVars().SetSystemVar(b.args[0].String(), b.args[1].String())
	if err != nil {
		return "nil", false, err
	}
	currentSysVals, succ := b.ctx.GetSessionVars().GetSystemVar(b.args[0].String())
	if !succ {
		return "nil", currentSysVals == "", nil
	}
	return currentSysVals, currentSysVals == "", nil
}

//pg_encoding_to_char

type pgEncodingToCharFunctionClass struct {
	baseFunctionClass
}

func (p *pgEncodingToCharFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, types.ETInt)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgEncodingToCharSig{bf}
	return sig, nil
}

type builtinPgEncodingToCharSig struct {
	baseBuiltinFunc
}

func (b *builtinPgEncodingToCharSig) Clone() builtinFunc {
	newSig := &builtinPgEncodingToCharSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgEncodingToCharSig) evalString(row chunk.Row) (string, bool, error) {
	id, isNull, err := b.args[0].EvalInt(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	charset := encodingToChar[id].name
	return charset, false, nil
}

type encodingToInt struct {
	name string
	code int
}

var encodingToChar = []encodingToInt{
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

//pg_has_database_privilege

type pgHasDatabasePrivilegeFunctionClass struct {
	baseFunctionClass
}

func (p *pgHasDatabasePrivilegeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	argTps := make([]types.EvalType, 0, 3)
	argTps = append(argTps, types.ETInt, types.ETString)
	if len(args) > 2 {
		idArgTp := make([]types.EvalType, 0, 1)
		idArgTp = append(idArgTp, types.ETInt)
		argTps = append(idArgTp, argTps...)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, argTps...)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgHasDatabasePrivilegeSig{bf}
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

func (b *builtinPgHasDatabasePrivilegeSig) evalString(row chunk.Row) (string, bool, error) {

	return "true", false, nil
}

//pg_has_table_privilege

type pgHasTablePrivilegeFunctionClass struct {
	baseFunctionClass
}

func (p *pgHasTablePrivilegeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	argsTps := make([]types.EvalType, 0, 3)
	argsTps = append(argsTps, types.ETString, types.ETString)
	if len(args) > 2 {
		argsTps = append(argsTps)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, argsTps...)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgHasTablePrivilegeSig{bf}
	return sig, nil
}

type builtinPgHasTablePrivilegeSig struct {
	baseBuiltinFunc
}

func (b *builtinPgHasTablePrivilegeSig) Clone() builtinFunc {
	newSig := &builtinPgHasTablePrivilegeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgHasTablePrivilegeSig) evalString(row chunk.Row) (string, bool, error) {

	return "true", false, nil
}

//pg_has_schema_privilege

type pgHasSchemaPrivilegeFunctionClass struct {
	baseFunctionClass
}

func (p *pgHasSchemaPrivilegeFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	argsTps := make([]types.EvalType, 0, 3)
	argsTps = append(argsTps, types.ETString, types.ETString)
	if len(args) > 2 {
		argsTps = append(argsTps)
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, argsTps...)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgHasSchemaPrivilegeSig{bf}
	return sig, nil
}

type builtinPgHasSchemaPrivilegeSig struct {
	baseBuiltinFunc
}

func (b *builtinPgHasSchemaPrivilegeSig) Clone() builtinFunc {
	newSig := &builtinPgHasSchemaPrivilegeSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgHasSchemaPrivilegeSig) evalString(row chunk.Row) (string, bool, error) {

	return "true", false, nil
}

// pg_is_in_recovery

type pgIsInRecoveryFunctionClass struct {
	baseFunctionClass
}

func (p *pgIsInRecoveryFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgIsInRecovery{bf}
	return sig, nil
}

type builtinPgIsInRecovery struct {
	baseBuiltinFunc
}

//暂且先这样处理,这里涉及到Pg内部的系统逻辑,这里没有合适的方法去实现
func (b *builtinPgIsInRecovery) evalString(row chunk.Row) (string, bool, error) {

	return "FALSE", false, nil
}

// pg_is_wal_replay_paused
type pgIsWalReplayPausedFunctionClass struct {
	baseFunctionClass
}

func (p *pgIsWalReplayPausedFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgIsWalReplayPaused{bf}
	return sig, nil
}

type builtinPgIsWalReplayPaused struct {
	baseBuiltinFunc
}

//todo 暂且先这样处理,这里涉及到Pg内部的系统逻辑,这里没有合适的方法去实现
func (b *builtinPgIsWalReplayPaused) evalString(row chunk.Row) (string, bool, error) {
	return "TRUE", false, nil
}

// pg_get_userbyid
type pgGetUserByIDFunctionClass struct {
	baseFunctionClass
}

func (p *pgGetUserByIDFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, types.ETInt)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgGetUserByIDSig{bf}
	return sig, nil
}

type builtinPgGetUserByIDSig struct {
	baseBuiltinFunc
}

func (b *builtinPgGetUserByIDSig) Clone() builtinFunc {
	newSig := &builtinPgGetUserByIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgGetUserByIDSig) evalString(row chunk.Row) (string, bool, error) {
	id, isNull, err := b.args[0].EvalInt(b.ctx, row)
	if isNull || err != nil {
		return "", isNull, err
	}
	userName := fmt.Sprintf("unknown(OID=%d)", id)
	if id == 10 {
		userName = b.ctx.GetSessionVars().User.Username
	}
	return userName, false, nil
}

// PgFuncShobjDescription
type pgShobjDescriptionFunctionClass struct {
	baseFunctionClass
}

func (p *pgShobjDescriptionFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, types.ETInt, types.ETString)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgShobjDescriptionSig{bf}
	return sig, nil
}

type builtinPgShobjDescriptionSig struct {
	baseBuiltinFunc
}

func (b *builtinPgShobjDescriptionSig) Clone() builtinFunc {
	newSig := &builtinPgGetUserByIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgShobjDescriptionSig) evalString(row chunk.Row) (string, bool, error) {

	return "[null]", false, nil
}

// PgFuncObjDescription
type pgObjDescriptionFunctionClass struct {
	baseFunctionClass
}

func (p *pgObjDescriptionFunctionClass) getFunction(ctx sessionctx.Context, args []Expression) (builtinFunc, error) {
	if err := p.verifyArgs(args); err != nil {
		return nil, err
	}
	bf, err := newBaseBuiltinFuncWithTp(ctx, p.funcName, args, types.ETString, types.ETInt)
	if err != nil {
		return nil, err
	}
	bf.tp.Charset, bf.tp.Collate = ctx.GetSessionVars().GetCharsetInfo()
	bf.tp.Flen = 64
	sig := &builtinPgObjDescriptionSig{bf}
	return sig, nil
}

type builtinPgObjDescriptionSig struct {
	baseBuiltinFunc
}

func (b *builtinPgObjDescriptionSig) Clone() builtinFunc {
	newSig := &builtinPgGetUserByIDSig{}
	newSig.cloneFrom(&b.baseBuiltinFunc)
	return newSig
}

func (b *builtinPgObjDescriptionSig) evalString(row chunk.Row) (string, bool, error) {
	return "[null]", false, nil
}
