package errors

type ErrorDef struct {
	Code     string
	Message  string
	HTTPCode int
}

var ERRORS = map[string]ErrorDef{
	"A001": {Code: "A001", Message: "Server error", HTTPCode: 500},
	"A002": {Code: "A002", Message: "Login error", HTTPCode: 500},
	"A003": {Code: "A003", Message: "Unauthorized", HTTPCode: 401},
	"A004": {Code: "A004", Message: "Forbidden role error", HTTPCode: 403},
	"A005": {Code: "A005", Message: "Create API token error", HTTPCode: 500},
	"A006": {Code: "A006", Message: "Delete API token error", HTTPCode: 500},
	"A009": {Code: "A009", Message: "Get Xray stats error", HTTPCode: 500},
	"A010": {Code: "A010", Message: "Failed to get system stats", HTTPCode: 500},
	"A011": {Code: "A011", Message: "Failed to get users stats", HTTPCode: 500},
	"A012": {Code: "A012", Message: "Failed to get inbound stats", HTTPCode: 500},
	"A013": {Code: "A013", Message: "Failed to get outbound stats", HTTPCode: 500},
	"A014": {Code: "A014", Message: "Failed to get inbound users", HTTPCode: 500},
	"A015": {Code: "A015", Message: "Failed to get inbounds stats", HTTPCode: 500},
	"A016": {Code: "A016", Message: "Failed to get outbounds stats", HTTPCode: 500},
	"A017": {Code: "A017", Message: "Failed to get combined stats", HTTPCode: 500},
}

const (
	CodeInternalServerError       = "A001"
	CodeLoginError                = "A002"
	CodeUnauthorized              = "A003"
	CodeForbiddenRoleError        = "A004"
	CodeCreateAPITokenError       = "A005"
	CodeDeleteAPITokenError       = "A006"
	CodeGetXrayStatsError         = "A009"
	CodeFailedToGetSystemStats    = "A010"
	CodeFailedToGetUsersStats     = "A011"
	CodeFailedToGetInboundStats   = "A012"
	CodeFailedToGetOutboundStats  = "A013"
	CodeFailedToGetInboundUsers   = "A014"
	CodeFailedToGetInboundsStats  = "A015"
	CodeFailedToGetOutboundsStats = "A016"
	CodeFailedToGetCombinedStats  = "A017"
)

func GetError(code string) (ErrorDef, bool) {
	e, ok := ERRORS[code]
	return e, ok
}
