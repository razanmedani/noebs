package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/prometheus/common/log"
	"gopkg.in/go-playground/validator.v8"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

func GetMainEngine() *gin.Engine {
	route := gin.Default()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterStructValidation(workingKeyStructValidators, WorkingKeyFields{})
	}

	route.HandleMethodNotAllowed = true

	route.POST("/workingKey", WorkingKey)
	route.POST("/cardTransfer", CardTransfer)
	route.POST("/purchase", Purchase)

	// This is like isAlive one...
	route.POST("/test", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"message": true})
	})
	return route
}

func main() {
	// Logging to a file.
	f, _ := os.Create("gin.log") // not sure whether this is the right place to do it. Maybe env vars?
	gin.DefaultWriter = io.MultiWriter(f)
	if env := os.Getenv("PORT"); env != "" {
		GetMainEngine().Run(env)
	} else {
		GetMainEngine().Run(":8080")
	}
}

func WorkingKey(c *gin.Context) {

	url := "path/to/ebs/endpoint" // EBS simulator endpoint url goes here.
	//FIXME instead of hardcoding it here, maybe offer it in the some struct that handles everything about the application configurations.
	// consume the request here and pass it over onto the EBS.
	// marshal the request
	// fuck. This shouldn't be here at all.

	var fields = WorkingKeyFields{}
	reqBody, err := ioutil.ReadAll(c.Request.Body)
	reader1 := ioutil.NopCloser(bytes.NewBuffer([]byte(reqBody)))
	reader2 := ioutil.NopCloser(bytes.NewBuffer([]byte(reqBody)))

	c.Request.Body = reader1
	if err != nil {
		fmt.Println("There's an error in nopclose init.")
		c.AbortWithError(500, err)
	}

	reqBodyErr := c.ShouldBindBodyWith(&fields, binding.JSON)
	switch {
	case reqBodyErr == nil:
		// request body was already consumed here. But the request
		// body was bounded to fields struct.
		c.Request.Body = reader2
		EBSHttpClient(url, c)
	case reqBodyErr == io.EOF:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you have not sent any request fields", "error": "empty_request_body"})
	case reqBodyErr != nil:
		c.JSON(http.StatusBadRequest, gin.H{"message": "Unknown client error", "error": "unknown_error"})
	}

	defer c.Request.Body.Close()

}

func Purchase(c *gin.Context) {

	url := "path/to/ebs/endpoint" // EBS simulator endpoint url goes here.
	//FIXME instead of hardcoding it here, maybe offer it in the some struct that handles everything about the application configurations.
	// consume the request here and pass it over onto the EBS.
	// marshal the request
	// fuck. This shouldn't be here at all.

	var fields = PurchaseFields{}
	reqBody, err := ioutil.ReadAll(c.Request.Body)
	reader1 := ioutil.NopCloser(bytes.NewBuffer([]byte(reqBody)))
	reader2 := ioutil.NopCloser(bytes.NewBuffer([]byte(reqBody)))

	c.Request.Body = reader1

	if err != nil {
		fmt.Println("There's an error in nopclose init.")
		c.AbortWithError(500, err)
	}

	reqBodyErr := c.ShouldBindBodyWith(&fields, binding.JSON)
	switch {
	case reqBodyErr == nil:
		// request body was already consumed here. But the request
		// body was bounded to fields struct.
		c.Request.Body = reader2
		EBSHttpClient(url, c)
	case reqBodyErr == io.EOF:
		c.JSON(http.StatusBadRequest, gin.H{"message": "you have not sent any request fields", "error": "empty_request_body"})
	case reqBodyErr != nil:
		c.JSON(http.StatusBadRequest, gin.H{"message": "Unknown client error", "error": "unknown_error"})
	}

	defer c.Request.Body.Close()

}

func CardTransfer(c *gin.Context) {

	url := "path/to/ebs/endpoint" // EBS simulator endpoint url goes here.
	//FIXME instead of hardcoding it here, maybe offer it in the some struct that handles everything about the application configurations.
	// consume the request here and pass it over onto the EBS.
	// marshal the request
	// fuck. This shouldn't be here at all.

	var fields = CardTransferFields{}

	reqBodyErr := c.ShouldBindBodyWith(&fields, binding.JSON)
	switch {

	case reqBodyErr != nil:
		err := Response{"error": gin.H{"message": "Unknown client error", "error": "unknown_error"}}
		c.JSON(http.StatusBadRequest, err)

	case reqBodyErr == io.EOF:
		err := Response{"error": gin.H{"message": "you have not sent any request fields", "error": "empty_request_body"}}
		c.JSON(http.StatusBadRequest, err)

	case reqBodyErr == nil:
		// request body was already consumed here. But the request
		// body was bounded to fields struct.
		// Now, decode the struct into a json, or bytes buffer.

		jsonBuffer, err := json.Marshal(fields)
		if err != nil{
			// there's an error in parsing the struct. Server error.
			err := Response{"error": gin.H{"message": err.Error(), "code": "parsing_error"}}
			log.Fatalf("there is an error. Request is %v", string(jsonBuffer))
			c.AbortWithStatusJSON(500, err)
		}

		code, res, err := EBSHttpClient2(url, bytes.NewBuffer(jsonBuffer))
		// {"ebs_error": "code": "message": ""}
		if err != nil {
			c.AbortWithStatusJSON(code, res)
		} else {
			// successful
			c.JSON(code, res)
		}
	}

	defer c.Request.Body.Close()

}

// a generic client for EBS's
func EBSHttpClient(url string, c *gin.Context) {

	verifyTLS := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	ebsClient := http.Client{
		Timeout:   30 * time.Second,
		Transport: verifyTLS,
	}

	reqHandler, err := http.NewRequest("POST", url, c.Request.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	reqHandler.Header.Set("Content-Type", "application/json")
	reqHandler.Header.Set("API-Key", "removeme") // For Morsal case only.
	// EBS doesn't impose any sort of API-keys or anything. Typical EBS.

	response, err := ebsClient.Do(reqHandler)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to reach EBS.",
			"code": err.Error()})
		return
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	// else, the response is really working!
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Unable to reach EBS.",
			"code": err.Error()})
		return
	}

	var ebsResponse map[string]string
	if err := json.Unmarshal(responseBody, &ebsResponse); err == nil {
		// there's no problem in Unmarshalling
		if responseCode, ok := ebsResponse["responseCode"]; ok { //Frankly, if we went this far it will work anyway.
			resCode, err := strconv.Atoi(string(responseCode))
			if err != nil {
				c.JSON(http.StatusInternalServerError, "There's a problem. Check again later.") //Fixme.
			}
			if resCode == 0 {
				// It's a successful transaction! Fuck it.
				c.JSON(http.StatusOK, responseBody)
			}
		} else {
			// Nope, it is not a successful transaction. You screwed.
			c.JSON(http.StatusBadRequest, responseBody) // return the response as it is.
		}
		// There's an error in Unmarshalling the responseBody. Highly unlikely though. I, screwed.
		c.JSON(http.StatusInternalServerError, "There's a problem. Check again later.") //Fixme.
	}

	defer response.Body.Close()

}

func EBSHttpClient2(url string, req io.Reader) (int, Response, error) {

	verifyTLS := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	ebsClient := http.Client{
		Timeout:   30 * time.Second,
		Transport: verifyTLS,
	}

	reqHandler, err := http.NewRequest("POST", url, req)
	if err != nil {
		fmt.Println(err.Error())
	}
	reqHandler.Header.Set("Content-Type", "application/json")
	reqHandler.Header.Set("API-Key", "removeme") // For Morsal case only.
	// EBS doesn't impose any sort of API-keys or anything. Typical EBS.

	response, err := ebsClient.Do(reqHandler)

	if err != nil {
		res := Response{"error": gin.H{"code": "internal_server_error", "message": "ebs_unreachable"}}
		return 500, res, fmt.Errorf("unable to reach ebs %v", err)

	}

	responseBody, err := ioutil.ReadAll(response.Body)

	defer response.Body.Close()

	if err != nil {
		return 500, Response{"error": gin.H{"code": "internal_server_error", "message": "ebs_unreachable"}}, fmt.Errorf("unable to reach ebs %v", err)

	}

	var ebsResponse map[string]string
	if err := json.Unmarshal(responseBody, &ebsResponse); err == nil {
		// there's no problem in Unmarshalling

		if responseCode, ok := ebsResponse["responseCode"]; ok { //Frankly, if we went this far it will work anyway.
			resCode, _ := strconv.Atoi(string(responseCode))

			if resCode == 0 {
				res := Response{"successful_transaction": gin.H{"message": ebsResponse, "code": resCode}}
				return 200, res, nil
			} else {
				// Error in the clients request
				res := Response{"EBS_error": gin.H{"message": ebsResponse, "code": resCode}}
				return 400, res, nil
			}

		} else {
			// there is no responseCode
			res := Response{"error": gin.H{"message": ebsResponse, "code": "unknown_error"}}
			return 400, res, nil
		}

	} else {
		res := Response{"error": gin.H{"message": "error parsing the request body", "code": "parsing_request_error"}}
		return 500, res, nil
	}

}
