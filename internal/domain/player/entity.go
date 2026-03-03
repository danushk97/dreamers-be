package player

import "time"

// Entity represents a player in the domain.
type Entity struct {
	ID                 string
	Name               string
	ImageURL           string
	Gender             string
	DateOfBirth        time.Time
	TNBAID             string
	District           string
	Phone              string
	RecentAchievements string
	TshirtSize         string
	AadharCardImageURL string
	CreatedAt          time.Time
}

// Valid genders.
const (
	GenderMale   = "MALE"
	GenderFemale = "FEMALE"
)

// Valid t-shirt sizes.
var ValidTshirtSizes = []string{"XS", "S", "M", "L", "XL", "XXL", "XXXL"}

// Tamil Nadu districts for validation.
var TamilNaduDistricts = []string{
	"Ariyalur", "Chengalpattu", "Chennai", "Coimbatore", "Cuddalore",
	"Dharmapuri", "Dindigul", "Erode", "Kallakurichi", "Kancheepuram",
	"Kanyakumari", "Karur", "Krishnagiri", "Madurai", "Mayiladuthurai",
	"Nagapattinam", "Namakkal", "Perambalur", "Pudukottai", "Ramanathapuram",
	"Ranipet", "Salem", "Sivaganga", "Thanjavur", "Theni", "Thiruvallur",
	"Thiruvarur", "Thoothukudi", "Tiruchirappalli", "Tirunelveli", "Tirupathur",
	"Tiruppur", "Tiruvannamalai", "Vellore", "Viluppuram", "Virudhunagar",
}
