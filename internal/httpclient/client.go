package httpsclient

import (
	"errors"   //check error types
	"net"      //detect network level error
	"net/http" //make http requests
	"strings"  //check error text
	"time"     //measure duration
	//if we dont import, go cant compile
)

// jus names with return values
func MakeRequest(url string) (statusCode int, duration time.Duration, err error) { //error reuturn nil or non-nil
  
	start := time.Now() //stopwatch start,how long someting takes

	//Go sends a GET request,
	//call the fuction and 2 values will come back - response and error
	resp, err := http.Get(url)

	//Current time - start time = duration
	duration = time.Since(start)

	if err != nil {
		// Handle real-world network errors cleanly
		var netErr *net.OpError //either nil or pointer to net.OpError value

		if errors.As(err, &netErr) { //errors are wrapped
			//as checks if err has a net.operror,if not , unwraps and checks again
			//if yes, assigns to netErr variable

			if netErr.Op == "dial" { //happed when tyrna connect, not when reading,writing or closing(server down, dns failed)
				if strings.Contains(netErr.Err.Error(), "refused") {
					return 0, duration, errors.New("connection refused")
				}
				if strings.Contains(netErr.Err.Error(), "no such host") { //domain naame doesnt exist,dns lookup failed
					return 0, duration, errors.New("no such host")
				}
			}
		}

		return 0, duration, err
	}

	defer resp.Body.Close()
	//run this line after function exit
	//end by body because http use network resourse, overtime program slows down due ot mormory leak
	return resp.StatusCode, duration, nil
}
