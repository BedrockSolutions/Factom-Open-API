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
	"fmt"
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
	es       service.EntryService
	cs       service.ChainService
	us       service.UserService
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

type EntryResponse struct {
	*model.Entry
	Links []string `json:"links" form:"links" query:"links" validate:""`
}

type ChainResponse struct {
	*model.Chain
	Links []string `json:"links" form:"links" query:"links" validate:""`
}

func NewApi(conf *config.Config, es service.EntryService, cs service.ChainService, us service.UserService) *Api {

	api := &Api{}

	api.validate = validator.New()

	api.conf = conf
	api.es = es
	api.cs = cs
	api.us = us

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
		user := api.us.GetUserByAccessToken(key)
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
	//api.Http.GET("/v1/entries/:entryhash", api.getEntry)

	// User
	api.Http.GET("/v1/user", api.getUser)

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
	resp := &ChainResponse{Chain: req}

	// validate ExtIDs, Content
	if err := api.validate.StructPartial(req, "ChainID"); err != nil {
		return api.ErrorResponse(err, c)
	}

	// search for chain into local db "chains"
	chain, err := api.cs.GetChain(req)
	if err != nil {
		return err
	}

	// if chain not found in local db, check on the blockchain
	if chain == nil {
		if !req.Exists() {
			err := fmt.Errorf("Chain %s does not exist", req.ChainID)
			return api.ErrorResponse(err, c)
		}
	}

	// if chain not found in local db (but found on blockchain!)
	// start goroutine to parse all entries
	if chain == nil {
		//go api.cs.ParseAllChainEntries(req)
	}

	// if chain not found in local db (but found on blockchain!) OR status != completed
	// check current status on the blockchain and update in local db
	if chain == nil || chain.Status != model.ChainCompleted {
		resp.Status, _ = req.GetStatusFromFactom()
		req.Status = resp.Status
		//		req.Synced =

		_, err = api.cs.CreateChain(req, api.user)
		if err != nil {
			return err
		}

		//		err = api.cs.BindChainToUser(req, api.user)
		//		if err != nil {
		//			return err
		//		}
	}

	return api.SuccessResponse(resp, c)

}

func (api *Api) createChain(c echo.Context) error {

	// Open API Chain struct
	req := &model.Chain{}

	// bind input data
	if err := c.Bind(req); err != nil {
		return api.ErrorResponse(err, c)
	}

	log.Debug("Validating input data")

	// validate ExtIDs, Content
	if err := api.validate.StructExcept(req, "ChainID"); err != nil {
		return api.ErrorResponse(err, c)
	}

	resp, err := api.cs.CreateChain(req, api.user)

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

	// validate ChainID, ExtID (if exists), Content
	if err := api.validate.StructExcept(req, "EntryHash"); err != nil {
		return api.ErrorResponse(err, c)
	}

	// check if entry fits into 10KB
	_, err := req.Fit10KB()
	if err != nil {
		return api.ErrorResponse(err, c)
	}

	// extend Entry
	resp := &EntryResponse{Entry: req}

	// calculate entryhash
	resp.EntryHash = req.Hash()
	resp.Status = "completed"

	// Create entry
	_, err = api.es.CreateEntry(req)
	if err != nil {
		return err
	}

	// send to factomd
	//	factom.ComposeTransaction()

	// increase user's usage
	api.user.Usage += 1
	api.us.UpdateUser(api.user)
	//	log.Info(api.user)

	return api.SuccessResponse(resp, c)
}

/*
func (api *Api) getEntry(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad request param `id`")
	}
	cat, err := api.es.GetEntry(id)
	if err != nil {
		return err
	}
	if cat == nil {
		return echo.NewHTTPError(http.StatusNotFound, "Category `id` = ", id, " not found")
	}
	return c.JSON(http.StatusOK, cat)
}
*/
