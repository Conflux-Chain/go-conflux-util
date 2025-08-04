package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const httpStatusInternalError = 600

type CsvData struct {
	Filename string
	Data     [][]string
	// bom head is included by default, set true to exclude it.
	ExcludeBOMHeader bool
}

func ResponseSuccess(c *gin.Context, data any) {
	if data == nil {
		c.JSON(http.StatusOK, ErrNil)
	} else {
		csvData, ok := data.(CsvData)
		if ok {
			ResponseCsv(c, csvData)
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

func ResponseCsv(c *gin.Context, data CsvData) {
	filename := data.Filename
	if !strings.HasSuffix(filename, ".csv") {
		filename = filename + ".csv"
	}

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", data.Filename))
	c.Writer.Header().Set("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if !data.ExcludeBOMHeader {
		//Write UTF-8 BOM header for Excel compatibility
		_ = writer.Write([]string{"\xEF\xBB\xBF"})
	}

	_ = writer.WriteAll(data.Data)
}
