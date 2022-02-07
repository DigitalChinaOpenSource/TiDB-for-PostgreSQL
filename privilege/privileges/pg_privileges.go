package privileges

import (
	"crypto/tls"
	"github.com/pingcap/tidb/parser/auth"
	"github.com/pingcap/tidb/util/logutil"
	"go.uber.org/zap"
)

// ConnectionVerification implements the Manager interface.
func (p *UserPrivileges) ConnectionVerification(user, host string, authentication, salt []byte, tlsState *tls.ConnectionState) (u string, h string, success bool) {
	if SkipWithGrant {
		p.user = user
		p.host = host
		success = true
		return
	}

	mysqlPriv := p.Handle.Get()
	record := mysqlPriv.connectionVerification(user, host)
	if record == nil {
		logutil.BgLogger().Error("get user privilege record fail",
			zap.String("user", user), zap.String("host", host))
		return
	}

	u = record.User
	h = record.Host

	globalPriv := mysqlPriv.matchGlobalPriv(user, host)
	if globalPriv != nil {
		if !p.checkSSL(globalPriv, tlsState) {
			logutil.BgLogger().Error("global priv check ssl fail",
				zap.String("user", user), zap.String("host", host))
			success = false
			return
		}
	}

	// Login a locked account is not allowed.
	locked := record.AccountLocked
	if locked {
		logutil.BgLogger().Error("try to login a locked account",
			zap.String("user", user), zap.String("host", host))
		success = false
		return
	}

	pwd := record.AuthenticationString
	if !p.isValidHash(record) {
		return
	}

	// empty password
	if len(pwd) == 0 && len(authentication) == 0 {
		p.user = user
		p.host = h
		success = true
		return
	}

	if len(pwd) == 0 || len(authentication) == 0 {
		return
	}

	hpwd, err := auth.DecodePasswordByMD5(pwd)
	if err != nil {
		logutil.BgLogger().Error("decode password string failed", zap.Error(err))
		return
	}

	pgSalt := [4]byte{salt[0], salt[1], salt[2], salt[3]}
	if !auth.CheckPassword(pgSalt, hpwd, string(authentication)) {
		return
	}

	p.user = user
	p.host = h
	success = true
	return
}

// NeedPassword returns true if password is required for the given user
func (p *UserPrivileges) NeedPassword(user, host string) bool {
	mysqlPriv := p.Handle.Get()
	record := mysqlPriv.connectionVerification(user, host)
	if record == nil {
		logutil.BgLogger().Error("get user privilege record fail",
			zap.String("user", user), zap.String("host", host))
		return false
	}

	if record.AuthenticationString != "" {
		return true
	}

	return false
}
