package handler

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestConsumeSMSOTP_CorrectCode(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:+971500000000").SetVal(1)
	mock.ExpectExpire("sms_otp_attempts:+971500000000", smsOTPAttemptWindow).SetVal(true)
	mock.ExpectGet("sms_otp:+971500000000").SetVal("123456|en")
	mock.ExpectDel("sms_otp:+971500000000").SetVal(1)
	mock.ExpectDel("sms_otp_attempts:+971500000000", "sms_rate:+971500000000").SetVal(2)

	res, err := consumeSMSOTP(context.Background(), rdb, "+971500000000", "123456")
	if err != nil || res != otpOK {
		t.Fatalf("want otpOK, got res=%v err=%v", res, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestConsumeSMSOTP_WrongCodeDoesNotConsume(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:p").SetVal(1)
	mock.ExpectExpire("sms_otp_attempts:p", smsOTPAttemptWindow).SetVal(true)
	mock.ExpectGet("sms_otp:p").SetVal("123456|en")

	res, err := consumeSMSOTP(context.Background(), rdb, "p", "000000")
	if err != nil || res != otpInvalid {
		t.Fatalf("want otpInvalid, got res=%v err=%v", res, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err) // no DEL expected: the real code must survive a wrong guess (until the cap)
	}
}

func TestConsumeSMSOTP_LocksAfterMaxAttempts(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:p").SetVal(smsOTPMaxAttempts + 1)
	mock.ExpectExpire("sms_otp_attempts:p", smsOTPAttemptWindow).SetVal(true)
	mock.ExpectDel("sms_otp:p").SetVal(1) // the valid code is destroyed, even if guessed right now

	res, err := consumeSMSOTP(context.Background(), rdb, "p", "123456")
	if err != nil || res != otpLocked {
		t.Fatalf("want otpLocked, got res=%v err=%v", res, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestConsumeSMSOTP_ExpiredCode(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:p").SetVal(1)
	mock.ExpectExpire("sms_otp_attempts:p", smsOTPAttemptWindow).SetVal(true)
	mock.ExpectGet("sms_otp:p").RedisNil()

	res, err := consumeSMSOTP(context.Background(), rdb, "p", "123456")
	if err != nil || res != otpInvalid {
		t.Fatalf("want otpInvalid, got res=%v err=%v", res, err)
	}
}

func TestConsumeSMSOTP_LostRaceIsRejected(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:p").SetVal(1)
	mock.ExpectExpire("sms_otp_attempts:p", smsOTPAttemptWindow).SetVal(true)
	mock.ExpectGet("sms_otp:p").SetVal("123456|en")
	mock.ExpectDel("sms_otp:p").SetVal(0) // a concurrent request already consumed it

	res, err := consumeSMSOTP(context.Background(), rdb, "p", "123456")
	if err != nil || res != otpInvalid {
		t.Fatalf("want otpInvalid, got res=%v err=%v", res, err)
	}
}

func TestConsumeSMSOTP_RedisErrorFailsClosed(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	mock.ExpectIncr("sms_otp_attempts:p").SetErr(redis.ErrClosed)

	res, err := consumeSMSOTP(context.Background(), rdb, "p", "123456")
	if err == nil || res == otpOK {
		t.Fatalf("redis failure must not authenticate: res=%v err=%v", res, err)
	}
}

func TestRegistrationAllowed(t *testing.T) {
	tests := []struct {
		name  string
		open  bool
		count int
		err   error
		want  bool
	}{
		{"open registration", true, 5, nil, true},
		{"closed, fresh install", false, 0, nil, true},
		{"closed, users exist", false, 1, nil, false},
		{"closed, count fails -> fail closed", false, 0, redis.ErrClosed, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registrationAllowed(tt.open, tt.count, tt.err); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
