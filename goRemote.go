package main

/**
	TODO:
	   * Current error handling is to panic and spit out the error. Probably should be a little more helpful than that to the user
	   * Currenly only SSH device is support. Not sure what other devices should be handled but tried to make the code easy to add additional devices
	   * Add ability to close a connection
	   * Add command line option to specify IP address
**/

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tkanos/gonfig"
	"gopkg.in/oleiade/reflections.v1"
)

func main() {
	deviceName := flag.String("device", "", "Device to connect to")
	flag.Parse()

	if *deviceName != "" {
		fmt.Printf("Connecting to device: %s", *deviceName)
	}

	home, err := os.UserHomeDir()
	configPath := fmt.Sprintf("%s/.goremote/config.json", home)
	fmt.Printf("Loading %s\n", configPath)

	configuration := Configuration{}
	err = gonfig.GetConf(configPath, &configuration)
	if err != nil {
		panic(err)
	}

	/**
	* Do Login
	**/
	fmt.Printf("Doing Login\n")

	loginResponse := &LoginResponse{}

	{
		loginRequest := &LoginRequest{
			Username: configuration.Username,
			Password: configuration.Password,
		}

		url := fmt.Sprintf("%s%s", configuration.BaseURL, LoginPath)
		req, err := buildRequest(url, loginRequest, configuration)

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer res.Body.Close()

		json.NewDecoder(res.Body).Decode(loginResponse)
	}

	configuration.Token = loginResponse.Token

	/**
	 * Get Device List
	 **/
	fmt.Printf("Loading Device List\n")

	deviceListResponse := &DeviceListResponse{}
	{
		url := fmt.Sprintf("%s%s", configuration.BaseURL, DeviceListPath)
		req, err := buildRequest(url, nil, configuration)

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer res.Body.Close()

		json.NewDecoder(res.Body).Decode(deviceListResponse)
	}

	if *deviceName == "" {
		fmt.Printf("\n\nFound %d devices:\n", len(deviceListResponse.Devices))
		for _, device := range deviceListResponse.Devices {
			fmt.Printf("\tCMD: goRemote -device=\"%s\"\n", device.DeviceAlias)
			fmt.Printf("\t\t%s_template\n", device.ServiceTitle)
			fmt.Printf("\t\t%s\n", device.DeviceAddress)
		}
	} else {
		for _, device := range deviceListResponse.Devices {
			if device.DeviceAlias == *deviceName {
				fmt.Printf("Connecting to \"%s\"\n", device.DeviceAlias)
				/**
				 * Generate connect string for device user passed in
				 **/
				connectResponse := &ConnectResponse{}
				{
					myIP := getMyIP()
					fmt.Printf("\tHostIP: %s\n\tDeviceAddress: %s\n", myIP, device.DeviceAddress)

					connectRequest := &ConnectRequest{
						Wait:          "true",
						DeviceAddress: device.DeviceAddress,
						HostIP:        myIP,
					}

					url := fmt.Sprintf("%s%s", configuration.BaseURL, ConnectPath)
					req, err := buildRequest(url, connectRequest, configuration)

					client := &http.Client{}
					res, err := client.Do(req)
					if err != nil {
						panic(err)
					}

					defer res.Body.Close()

					json.NewDecoder(res.Body).Decode(connectResponse)

					template := fmt.Sprintf("%sTemplate", device.ServiceTitle)
					value, err := reflections.GetField(configuration, template)
					if err != nil {
						panic(err)
					}

					cmd := fmt.Sprintf("%s", value)

					// TODO: I guess there should be a way to "close" the connection with the connection-id
					//    https://docs.remote.it/api-reference/devices/device-connect-stop
					fmt.Printf("\nConection-Id: %s\n", connectResponse.ConnectionId)

					result := strings.Replace(cmd, "${PORT}", connectResponse.Connection.ProxyPort, -1)
					result = strings.Replace(result, "${HOST}", connectResponse.Connection.ProxyServer, -1)

					fmt.Printf("%s\n", result)
				}
			}
		}
	}
}

/**
 * This is some random service so may need a plug in a different one later
 **/
func getMyIP() string {
	res, err := http.Get("http://icanhazip.com")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		bodyString := string(bodyBytes)

		return strings.TrimSpace(bodyString)
	}

	return ""
}

func buildRequest(url string, obj interface{}, configuration Configuration) (*http.Request, error) {
	// Test Login Request
	buf := new(bytes.Buffer)

	if obj != nil {
		json.NewEncoder(buf).Encode(obj)
		req, err := http.NewRequest("POST", url, buf)
		req.Header.Set("developerkey", configuration.DeveloperKey)
		if configuration.Token != "" {
			//fmt.Printf("\n***Set Token***\n")
			req.Header.Set("token", configuration.Token)
		}
		return req, err
	} else {
		req, err := http.NewRequest("GET", url, buf)
		req.Header.Set("developerkey", configuration.DeveloperKey)
		if configuration.Token != "" {
			//fmt.Printf("\n***Set Token***\n")
			req.Header.Set("token", configuration.Token)
		}
		return req, err
	}
}

const LoginPath = "/apv/v27/user/login"
const DeviceListPath = "/apv/v27/device/list/all"
const ConnectPath = "/apv/v27/device/connect"

// Configuration - application state
type Configuration struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	DeveloperKey string `json:"developerkey"`
	BaseURL      string `json:"remoteit_baseurl"`
	Token        string `json:"token"`
	SSHTemplate  string `json:"SSH_template"`
}

// LoginRequest - This endpoint allows you to login to your
// remote.it account and get a authentication token which is
// required for all other authenticated APIs.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse - Login respones from server
type LoginResponse struct {
	Status            string `json:"status"`
	Token             string `json:"token"`
	TokenIndex        string `json:"token_index"`
	Email             string `json:"email"`
	GUID              string `json:"guid"`
	ServiceToken      string `json:"service_token"`
	ServiceLevel      string `json:"service_level"`
	StoragePlan       string `json:"storage_plan"`
	SecondaryAuth     string `json:"secondary_auth"`
	AuthToken         string `json:"auth_token"`
	AuthExpiration    int    `json:"auth_expiration"`
	ServiceAuthhash   string `json:"service_authhash"`
	CommericalSetting string `json:"commerical_setting"`
	Apikey            string `json:"apikey"`
	DeveloperKey      string `json:"developer_key"`
	Language          string `json:"language"`
	DeveloperPlan     string `json:"developer_plan"`
	PortalPlan        string `json:"portal_plan"`
	PortalPlanExpires string `json:"portal_plan_expires"`
	ServiceFeatures   string `json:"service_features"`
	Announcements     []struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		URL         string `json:"url"`
		Title       string `json:"title"`
		AckRequired string `json:"ack_required"`
		Created     string `json:"created"`
	} `json:"announcements"`
	MemberSince   string `json:"member_since"`
	Index         string `json:"index"`
	PubsubChannel string `json:"pubsub_channel"`
	AwsIdentity   string `json:"aws_identity"`
}

// DeviceListResponse - List of devices in account from server
type DeviceListResponse struct {
	Status       string   `json:"status"`
	CacheExpires int      `json:"cache_expires"`
	Length       int      `json:"length"`
	Devices      []Device `json:"devices"`
}

// Device - A Device
type Device struct {
	DeviceAddress  string        `json:"deviceaddress"`
	DeviceAlias    string        `json:"devicealias"`
	OwnerUsername  string        `json:"ownerusername"`
	DeviceType     string        `json:"devicetype"`
	DeviceState    string        `json:"devicestate"`
	DeviceLastIP   string        `json:"devicelastip"`
	LastInternalIP string        `json:"lastinternalip"`
	ServiceTitle   string        `json:"servicetitle"`
	WebEnabled     string        `json:"webenabled"`
	WebURI         []interface{} `json:"weburi"`
	LocalURL       string        `json:"localurl"`
	Georegion      string        `json:"georegion"`
	WebViewerURL   []interface{} `json:"webviewerurl"`
	LastContacted  time.Time     `json:"lastcontacted"`
	CreateDate     time.Time     `json:"createdate"`
	Shared         string        `json:"shared"`
	Scripting      bool          `json:"scripting"`
}

// ConnectRequest - A connect request to the server
type ConnectRequest struct {
	Wait          string `json:"wait"`
	DeviceAddress string `json:"deviceaddress"`
	HostIP        string `json:"hostip"`
}

// ConnectResponse - A connect response from the server
type ConnectResponse struct {
	Status     string `json:"status"`
	Connection struct {
		ConnectionOverridden string        `json:"connectionOverridden"`
		DeviceAddress        string        `json:"deviceaddress"`
		ExpirationSec        string        `json:"expirationsec"`
		ImageIntervalms      string        `json:"imageintervalms"`
		PreviousConnection   string        `json:"previousConnection"`
		Proxy                string        `json:"proxy"`
		ProxyPort            string        `json:"proxyport"`
		ProxyServer          string        `json:"proxyserver"`
		Requested            string        `json:"requested"`
		Status               string        `json:"status"`
		Streamscheme         []interface{} `json:"streamscheme"`
		Streamuri            []interface{} `json:"streamuri"`
		URL                  []interface{} `json:"url"`
		RequestedAt          time.Time     `json:"requestedAt"`
	} `json:"connection"`
	ConnectionId string `json:"connectionid"`
}
