package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/luraproject/lura/v2/proxy"
)

func main() {}

func init() {
	fmt.Println(string(ModifierRegisterer), "loaded!!!")
}

// ModifierRegisterer is the symbol the plugin loader will be looking for. It must
// implement the plugin.Registerer interface
// https://github.com/luraproject/lura/blob/master/proxy/plugin/modifier.go#L71
var ModifierRegisterer = registerer("krakend-debugger")

type registerer string

// RegisterModifiers is the function the plugin loader will call to register the
// modifier(s) contained in the plugin using the function passed as argument.
// f will register the factoryFunc under the name and mark it as a request
// and/or response modifier.
func (r registerer) RegisterModifiers(f func(
	name string,
	factoryFunc func(map[string]interface{}) func(interface{}) (interface{}, error),
	appliesToRequest bool,
	appliesToResponse bool,
)) {
	f(string(r)+"-request", r.requestDump, true, false)
	f(string(r)+"-response", r.responseDump, false, true)
	fmt.Println(string(r), "registered!!!")
}

// RequestWrapper is an interface for passing proxy request between the krakend pipe
// and the loaded plugins
type RequestWrapper interface {
	Params() map[string]string
	Headers() map[string][]string
	Body() io.ReadCloser
	Method() string
	URL() *url.URL
	Query() url.Values
	Path() string
}

// ResponseWrapper is an interface for passing proxy response between the krakend pipe
// and the loaded plugins
type ResponseWrapper interface {
	Data() map[string]interface{}
	Io() io.Reader
	IsComplete() bool
	Metadata() proxy.ResponseWrapper
}

type JwtClaims struct {
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
	Jti string `json:"jti"`
	Iss string `json:"iss"`
	Aud string `json:"aud"`
	Sub string `json:"sub"`
	Typ string `json:"typ"`
	Azp string `json:"azp"`
	SessionState string `json:"session_state"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess struct {
		Account struct {
			Roles []string `json:"roles"`
		} `json:"account"`
	}
	Scope string `json:"scope"`
	Sid string `json:"sid"`
	EmailVerified bool `json:"email_verified"`
	UserType string `json:"user_type"`
	UserID string `json:"user_id"`
	Name string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Email string `json:"email"`
}

func (c *JwtClaims) Valid() error {
	// Token'in süresinin dolup dolmadığını kontrol etmek için gerekli kodu burada ekleyebilirsiniz
	// Örneğin, şu anki tarihin token'in son kullanma tarihinden önce olup olmadığını kontrol edebilirsiniz.
	// Bu kontrolü yaparken dönecek hata değeri nil olmalıdır eğer token geçerli ise.
	// Eğer token süresi geçmişse, bir hata döndürebilirsiniz, örneğin:
	// return errors.New("Token süresi geçmiş")
	return nil
}

func (r registerer) requestDump(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	// check the cfg. If the modifier requires some configuration,
	// it should be under the name of the plugin.
	// ex: if this modifier required some A and B config params
	/*
	   "extra_config":{
	       "plugin/req-resp-modifier":{
	           "name":["krakend-debugger"],
	           "krakend-debugger":{
	               "A":"foo",
	               "B":42
	           }
	       }
	   }
	*/

	// return the modifier
	return func(input interface{}) (interface{}, error) {
		req, ok := input.(RequestWrapper)
		if !ok {
			return nil, errors.New("request:something went wrong")
		}
		// fmt.Println("req:", req)

		headerToken := req.Headers()["Authorization"]
		if len(headerToken) == 0 {
			return input, errors.New("request:token not found")
		}

		t := strings.Trim(headerToken[0][7:], " ")

		fmt.Println(t)

		token, _ := jwt.ParseWithClaims(t, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte{}, nil
		})

		claims, ok:= token.Claims.(*JwtClaims)

		if !ok {
			return input, errors.New("request: jwt cliams bind error")
		}

		fmt.Println("claims:", claims)

		log := fmt.Sprintf("Url: %s, Method: %s, UserID: %s, ReqBody: %s, Agent: %s", req.URL(), req.Method(), claims.UserID, req.Body(), req.Headers()["User-Agent"])

		fmt.Println(log)

		return input, nil
	}
}

func (r registerer) responseDump(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	// check the cfg. If the modifier requires some configuration,
	// it should be under the name of the plugin.
	// ex: if this modifier required some A and B config params
	/*
	   "extra_config":{
	       "plugin/req-resp-modifier":{
	           "name":["krakend-debugger"],
	           "krakend-debugger":{
	               "A":"foo",
	               "B":42
	           }
	       }
	   }
	*/

	// return the modifier
	return func(input interface{}) (interface{}, error) {
		resp, ok := input.(ResponseWrapper)
		if !ok {
			return nil, errors.New("response:something went wrong")
		}

		fmt.Println("data:", resp.Data())
		fmt.Println("is complete:", resp.IsComplete())
		fmt.Println("headers:", resp.Metadata().Headers())
		fmt.Println("status code:", resp.Metadata().StatusCode())

		return input, nil
	}
}