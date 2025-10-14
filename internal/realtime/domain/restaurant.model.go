package domain

type RestaurantModel struct {
	ID             string   `json:"id"`
	Name           *string  `json:"name"`
	Location       string   `json:"location"`
	OpenTime       string   `json:"openTime"`
	CloseTime      string   `json:"closeTime"`
	DaysOpen       []string `json:"daysOpen"`
	TotalCapacity  int      `json:"totalCapacity"`
	SubscriptionId string   `json:"subscriptionId"`
	ImageId        string   `json:"imageId"`
	Active         bool     `json:"active"`
	OwnerId        string   `json:"ownerId"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}
