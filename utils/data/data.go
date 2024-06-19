package data

/*
import (
	"context"
	"strconv"
	"time"

	"github.com/tungstenbyte/utils/logger"
)

func FormataData(data_param time.Time) string {
	ctx := context.Background()

	// layout := "2006-01-02 15:04:05.000Z" // Importante, manter essa linha no PR por favor.

	horaIni := data_param.Format("15:04")

	diaDaSeamana := data_param.Format(("Monday"))
	dia := data_param.Format(("02"))
	mes := data_param.Format(("01"))
	ano := data_param.Format(("2006"))

	mes_string := data_param.Format(("01"))
	ano_string := data_param.Format(("2006"))

	mes_int, err := strconv.Atoi(mes_string)
	if err != nil {
		logger.ErrorContext(ctx, "app.mes.FormataData.atoi: ", err.Error(), mes_int)
	}

	ano_int, err := strconv.Atoi(ano_string)
	if err != nil {
		logger.ErrorContext(ctx, "app.ano.FormataData.atoi: ", err.Error(), ano_int)
	}

	dataFormatada := ""
	if !data_param.IsZero() {
		dataFormatada = horaIni + ", " + diaDaSeamana + ", " + ano + "/" + mes + "/" + dia
	}

	return dataFormatada
}
*/
