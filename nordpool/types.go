package nordpool

import "time"

const API_URL = "https://www.nordpoolgroup.com/api"

type EnergyPrice struct {
	Hour  time.Time
	Price float64
}

type nordpool struct {
	Data     data         `json:"data"`
	CacheKey string       `json:"cacheKey"`
	Conf     *interface{} `json:"conf"`
	Header   *interface{} `json:"header"`
	EndDate  *string      `json:"endDate"`
	Currency string       `json:"currency"`
	PageID   int          `json:"pageId"`
}

type data struct {
	Rows                      []row         `json:"Rows"`
	IsDivided                 bool          `json:"IsDivided"`
	SectionNames              []string      `json:"SectionNames"`
	EntityIDs                 []string      `json:"EntityIDs"`
	DataStartDate             string        `json:"DataStartdate"`
	DataEndDate               string        `json:"DataEnddate"`
	MinDateForTimeScale       string        `json:"MinDateForTimeScale"`
	AreaChanges               []interface{} `json:"AreaChanges"`
	Units                     []string      `json:"Units"`
	LatestResultDate          string        `json:"LatestResultDate"`
	ContainsPreliminaryValues bool          `json:"ContainsPreliminaryValues"`
	ContainsExchangeRates     bool          `json:"ContainsExchangeRates"`
	ExchangeRateOfficial      string        `json:"ExchangeRateOfficial"`
	ExchangeRatePreliminary   string        `json:"ExchangeRatePreliminary"`
	ExchangeUnit              string        `json:"ExchangeUnit"`
	DateUpdated               string        `json:"DateUpdated"`
	CombinedHeadersEnabled    bool          `json:"CombinedHeadersEnabled"`
	DataType                  int           `json:"DataType"`
	TimeZoneInformation       int           `json:"TimeZoneInformation"`
}

type row struct {
	Columns         []column     `json:"Columns"`
	Name            string       `json:"Name"`
	StartTime       string       `json:"StartTime"`
	EndTime         string       `json:"EndTime"`
	DateTimeForData string       `json:"DateTimeForData"`
	DayNumber       int          `json:"DayNumber"`
	StartTimeDate   string       `json:"StartTimeDate"`
	IsExtraRow      bool         `json:"IsExtraRow"`
	IsNtcRow        bool         `json:"IsNtcRow"`
	EmptyValue      string       `json:"EmptyValue"`
	Parent          *interface{} `json:"Parent"`
}

type column struct {
	Index                            int          `json:"Index"`
	Scale                            int          `json:"Scale"`
	SecondaryValue                   *interface{} `json:"SecondaryValue"`
	IsDominatingDirection            bool         `json:"IsDominatingDirection"`
	IsValid                          bool         `json:"IsValid"`
	IsAdditionalData                 bool         `json:"IsAdditionalData"`
	Behavior                         int          `json:"Behavior"`
	Name                             string       `json:"Name"`
	Value                            string       `json:"Value"`
	GroupHeader                      string       `json:"GroupHeader"`
	DisplayNegativeValueInBlue       bool         `json:"DisplayNegativeValueInBlue"`
	CombinedName                     string       `json:"CombinedName"`
	DateTimeForData                  string       `json:"DateTimeForData"`
	DisplayName                      string       `json:"DisplayName"`
	DisplayNameOrDominatingDirection string       `json:"DisplayNameOrDominatingDirection"`
	IsOfficial                       bool         `json:"IsOfficial"`
	UseDashDisplayStyle              bool         `json:"UseDashDisplayStyle"`
}
