package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const httpStatusInternalError = 600

type CsvData struct {
	Filename string
	Data     [][]string
}

func ResponseSuccess(c *gin.Context, data any) {
	if data == nil {
		c.JSON(http.StatusOK, ErrNil)
	} else {
		csvData, ok := data.(CsvData)
		if ok {
			ResponseCsv(c, csvData.Filename, csvData.Data)
		} else {
			c.JSON(http.StatusOK, ErrNil.WithData(data))
		}
	}
}

func ResponseError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *BusinessError:
		c.JSON(http.StatusOK, e)
	case validator.ValidationErrors: // binding error
		c.JSON(http.StatusOK, ErrValidation(e))
	default:
		// internal server error
		c.JSON(httpStatusInternalError, ErrInternal(e))
	}
}

func ResponseCsv(c *gin.Context, filename string, content [][]string) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	writer.WriteAll(content)

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v.csv", filename))
	c.Writer.Header().Set("Content-Type", "text/csv")
	c.Writer.Write(buf.Bytes())
}
