// Factom-Open-API
// Version: 1.0
// Schemes: http
// Host: localhost
// BasePath: /v1
// Consumes:
// - application/json
// - application/x-www-form-urlencoded
// - multipart/form-data
// Produces:
// - application/json
// Contact: team@de-facto.pro
// swagger:meta
package api

import (
	"encoding/json"
	"fmt"
	"github.com/FactomProject/factom"
	"net/http"
	"strconv"

	"github.com/DeFacto-Team/Factom-Open-API/config"
	"github.com/DeFacto-Team/Factom-Open-API/model"
	"github.com/DeFacto-Team/Factom-Open-API/service"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"gopkg.in/go-playground/validator.v9"
)

type Api struct {
	Http     *echo.Echo
	conf     *config.Config
	service  service.Service
	apiInfo  ApiInfo
	validate *validator.Validate
	user     *model.User
}

type ApiInfo struct {
	Address string
	MW      []string
}

type ErrorResponse struct {
	Result bool   `json:"result"`
	Code   int    `json:"code"`
	Error  string `json:"error"`
}

type SuccessResponse struct {
	Result interface{} `json:"result"`
}

func NewApi(conf *config.Config, s service.Service) *Api {

	api := &Api{}

	api.validate = validator.New()

	api.conf = conf
	api.service = s

	api.Http = echo.New()
	api.Http.Logger.SetLevel(log.Lvl(conf.LogLevel))
	api.apiInfo.Address = ":" + strconv.Itoa(api.conf.Api.HttpPort)
	api.Http.HideBanner = true
	api.Http.Pre(middleware.RemoveTrailingSlash())

	if conf.Api.Logging {
		api.Http.Use(middleware.Logger())
		api.apiInfo.MW = append(api.apiInfo.MW, "Logger")
	}

	api.Http.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		user := api.service.GetUserByAccessToken(key)
		if user != nil && user.Status == 1 {
			api.user = user
			return true, nil
		}
		return false, fmt.Errorf("User not found")
	}))

	api.apiInfo.MW = append(api.apiInfo.MW, "KeyAuth")

	api.Http.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: conf.GzipLevel,
	}))

	// Status
	api.Http.GET("/v1", api.index)

	// API specification
	api.Http.Static("/v1/spec", "spec")

	// Chains
	api.Http.POST("/v1/chains", api.createChain)
	api.Http.GET("/v1/chains/:chainid", api.getChain)

	// Entries
	api.Http.POST("/v1/entries", api.createEntry)
	api.Http.GET("/v1/entries/:entryhash", api.getEntry)

	// User
	api.Http.GET("/v1/user", api.getUser)

	// Direct factomd call
	api.Http.POST("/v1/factomd/:method", api.factomd)

	return api
}

// Start API server
func (api *Api) Start() error {
	return api.Http.Start(":" + strconv.Itoa(api.conf.Api.HttpPort))
}

// Returns API information
func (api *Api) GetApiInfo() ApiInfo {
	return api.apiInfo
}

// Returns API user info
func (api *Api) getUser(c echo.Context) error {
	return c.JSON(http.StatusOK, &api.user)
}

// Returns API specification (Swagger)
func (api *Api) index(c echo.Context) error {
	return c.Redirect(http.StatusMovedPermanently, "/spec/api.json")
}

func (api *Api) spec(c echo.Context) error {
	return c.Inline("spec/api.json", "api.json")
}

// Success API response
func (api *Api) SuccessResponse(res interface{}, c echo.Context) error {
	return c.JSON(http.StatusOK, &SuccessResponse{Result: res})
}

// Custom API response in case of error
func (api *Api) ErrorResponse(err error, c echo.Context) error {
	resp := &ErrorResponse{
		Result: false,
		Code:   http.StatusBadRequest,
		Error:  err.Error(),
	}
	log.Error(err.Error())
	api.Http.Logger.Error(resp.Error)
	return c.JSON(resp.Code, resp)
}

// API functions

func (api *Api) getChain(c echo.Context) error {

	req := &model.Chain{ChainID: c.Param("chainid")}

	log.Debug("Validating input data")

	// validate ExtIDs, Content
	if err := api.validate.StructPartial(req, "ChainID"); err != nil {
		return api.ErrorResponse(err, c)
	}

	resp := api.service.GetChain(req, api.user)

	if resp == nil {
		return api.ErrorResponse(fmt.Errorf("Chain %s does not exist", req.ChainID), c)
	}

	return api.SuccessResponse(resp, c)

}

func (api *Api) createChain(c echo.Context) error {

	// Open API Chain struct
	req := &model.Chain{}

	req.Content = c.FormValue("content")

	// bind input data
	if err := c.Bind(req); err != nil {
		return api.ErrorResponse(err, c)
	}

	log.Debug("Validating input data")

	// validate ExtIDs, Content
	if err := api.validate.StructExcept(req, "ChainID"); err != nil {
		return api.ErrorResponse(err, c)
	}

	resp, err := api.service.CreateChain(req, api.user)

	if err != nil {
		return api.ErrorResponse(err, c)
	}

	return api.SuccessResponse(resp, c)
}

// swagger:operation POST /entries createEntry
// ---
// description: Create entry in chain
// parameters:
// - name: chainid
//   in: body
//   description: Chain ID of the Factom chain where to add new entry.
//   required: true
//   type: string
// - name: content
//   in: body
//   description: The content of new entry.
//   required: true
//   type: string
// - name: extids
//   in: body
//   description: One or many external ids identifying new entry. Should be sent as array of strings.
//   required: false
//   type: array
func (api *Api) createEntry(c echo.Context) error {

	// Open API Entry struct
	req := &model.Entry{}

	// bind input data
	if err := c.Bind(req); err != nil {
		return api.ErrorResponse(err, c)
	}

	log.Debug("Validating input data")

	// validate ChainID, ExtID (if exists), Content
	if err := api.validate.StructExcept(req, "EntryHash"); err != nil {
		return api.ErrorResponse(err, c)
	}

	// Create entry
	resp, err := api.service.CreateEntry(req, api.user)

	if err != nil {
		return api.ErrorResponse(err, c)
	}

	return api.SuccessResponse(resp, c)
}

func (api *Api) getEntry(c echo.Context) error {

	req := &model.Entry{EntryHash: c.Param("entryhash")}

	log.Debug("Validating input data")

	// validate ExtIDs, Content
	if err := api.validate.StructPartial(req, "EntryHash"); err != nil {
		return api.ErrorResponse(err, c)
	}

	resp := api.service.GetEntry(req, api.user)

	if resp == nil {
		return api.ErrorResponse(fmt.Errorf("Entry %s does not exist", req.EntryHash), c)
	}

	return api.SuccessResponse(resp, c)

}

func (api *Api) factomd(c echo.Context) error {

	var params interface{}

	if c.FormValue("params") != "" {
		log.Info(c.FormValue("params"))
		err := json.Unmarshal([]byte(c.FormValue("params")), &params)
		if err != nil {
			return api.ErrorResponse(err, c)
		}
	}

	request := factom.NewJSON2Request(c.Param("method"), 0, params)

	resp, err := factom.SendFactomdRequest(request)
	if err != nil {
		return api.ErrorResponse(err, c)
	}

	if resp.Error != nil {
		return api.ErrorResponse(resp.Error, c)
	}

	return api.SuccessResponse(resp.Result, c)

}
