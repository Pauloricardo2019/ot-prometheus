package data

import (
	"log"
	"time"
)

const (
	DataStatementUTC         = "2006-01-02T00:00:00-03:00"
	DataStatement            = "2006-01-02T00:00:00"
	DataStatementTimeZone    = "2006-01-02T00:00:00Z"
	DataStatementUTCTimeZone = "2006-01-02T03:00:00Z"
	Desconhecido             = "2006-01-02T00:00:00.000"
	layoutBR                 = "02/01/2006"
	layoutUS                 = "01/02/2006"
	layoutAnoMesDiaBarra     = "2006/01/02"
	layoutAnoMesDiatraco     = "2006-01-02"
)

func DataPadrao(dt string) string {
	if t := TryParse(dt); t != nil {
		if t.IsZero() {
			return "-"
		}
		return t.Format(DataStatement)
	}
	return "-"
}

func TryParse(dt string) *time.Time {
	if t, err := time.Parse(DataStatementUTC, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(DataStatement, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(DataStatementTimeZone, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(DataStatementUTCTimeZone, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(layoutBR, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(layoutUS, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(Desconhecido, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(layoutAnoMesDiaBarra, dt); err == nil {
		return &t
	}
	if t, err := time.Parse(layoutAnoMesDiatraco, dt); err == nil {
		return &t
	}
	log.Println("data inválida para ser tratada pelo tryparse : ", dt)
	retorna := time.Time{}
	return &retorna
}
