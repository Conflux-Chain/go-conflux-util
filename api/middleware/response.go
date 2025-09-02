package middleware

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const HttpStatusInternalError = 600

type CsvData struct {
	Filename string
	Data     [][]string
	// bom head is included by default, set true to exclude it.
	ExcludeBOMHeader bool
}

func ResponseSuccess(c *gin.Context, data any) {
	if data == nil {
		c.JSON(http.StatusOK, api.ErrNil)
	} else if csvData, ok := data.(CsvData); ok {
		ResponseCsv(c, csvData)
	} else {
		c.JSON(http.StatusOK, api.ErrNil.WithData(data))
	}
}

func ResponseError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *api.BusinessError:
		c.JSON(http.StatusOK, e)
	case validator.ValidationErrors: // binding error
		c.JSON(http.StatusOK, api.ErrValidation(e))
	default:
		// internal server error
		c.JSON(HttpStatusInternalError, api.ErrInternal(e))
	}
}

func ResponseCsv(c *gin.Context, data CsvData) {
	filename := data.Filename
	if !strings.HasSuffix(filename, ".csv") {
		filename = filename + ".csv"
	}

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", filename))
	c.Writer.Header().Set("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if !data.ExcludeBOMHeader {
		//Write UTF-8 BOM header for Excel compatibility
		_ = writer.Write([]string{"\xEF\xBB\xBF"})
	}

	_ = writer.WriteAll(data.Data)
}
