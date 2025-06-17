package helper

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/savioruz/goth/pkg/constant"
	"math/big"
	"time"
)

const (
	x = 10
)

// PgString converts a string to pgtype.Text
func PgString(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

// PgInt64 converts an int64 to pgtype.Numeric
func PgInt64(i int64) pgtype.Numeric {
	bigInt := new(big.Int).SetInt64(i)

	return pgtype.Numeric{
		Int:   bigInt,
		Valid: true,
	}
}

// Int64FromPg converts a pgtype.Numeric to an int64
func Int64FromPg(n pgtype.Numeric) int64 {
	if !n.Valid || n.Int == nil {
		return 0
	}

	if n.Exp != 0 {
		// Create a copy of the number to work with
		result := new(big.Int).Set(n.Int)

		// If Exp is negative, divide by 10^(-Exp)
		if n.Exp < 0 {
			divisor := new(big.Int).Exp(big.NewInt(x), big.NewInt(int64(-n.Exp)), nil)
			result = result.Div(result, divisor)
		} else {
			// If Exp is positive, multiply by 10^Exp
			multiplier := new(big.Int).Exp(big.NewInt(x), big.NewInt(int64(n.Exp)), nil)
			result = result.Mul(result, multiplier)
		}

		return result.Int64()
	}

	return n.Int.Int64()
}

// PgUUID converts a string UUID to pgtype.UUID
func PgUUID(id string) pgtype.UUID {
	var uuid pgtype.UUID

	err := uuid.Scan(id)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}

	return uuid
}

// PgDate converts a string date to pgtype.Date
func PgDate(date string) pgtype.Date {
	var pgDate pgtype.Date

	err := pgDate.Scan(date)
	if err != nil {
		return pgtype.Date{Valid: false}
	}

	return pgDate
}

// PgTimeFromString converts a time string (format "15:04") to pgtype.Time
func PgTimeFromString(timeStr string) (pgtype.Time, error) {
	parsedTime, err := time.Parse(constant.HoursFormat, timeStr)
	if err != nil {
		return pgtype.Time{Valid: false}, err
	}

	return pgtype.Time{
		Microseconds: int64((parsedTime.Hour()*constant.SecondsPerHour + parsedTime.Minute()*constant.MinutesPerHour) * constant.MicrosecondsPerSec),
		Valid:        true,
	}, nil
}

// PgTimeFromTime converts a time.Time object to pgtype.Time
func PgTimeFromTime(t time.Time) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64((t.Hour()*constant.SecondsPerHour + t.Minute()*constant.MinutesPerHour) * constant.MicrosecondsPerSec),
		Valid:        true,
	}
}

// TimeFromString converts a time string (format "15:04") to a time.Time object
func TimeFromString(s string) time.Time {
	t, err := time.Parse(constant.HoursFormat, s)
	if err != nil {
		return time.Time{}
	}

	now := time.Now()

	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
}

// TimeToString converts a time.Time object to a string in the format "15:04"
func TimeToString(t time.Time) string {
	return t.Format(constant.HoursFormat)
}

func PgTimeToString(t pgtype.Time) (string, error) {
	if !t.Valid {
		return "", nil
	}

	totalSeconds := t.Microseconds / constant.MicrosecondsPerSec
	hours := totalSeconds / constant.SecondsPerHour
	minutes := (totalSeconds % constant.SecondsPerHour) / constant.MinutesPerHour

	return time.Date(0, 1, 1, int(hours), int(minutes), 0, 0, time.Local).Format(constant.HoursFormat), nil
}

// PgTimestamp converts a time.Time object to pgtype.Timestamp
func PgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:             t,
		InfinityModifier: 0,
		Valid:            true,
	}
}
