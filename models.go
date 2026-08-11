package xivmarketgo

import "encoding/json"

type CurrentlyShown struct {
	// The item ID.
	ItemId int32 `json:"itemID"`
	// The world ID, if applicable.
	WorldId int32 `json:"worldID"`
	// The last upload time for this endpoint, in milliseconds since the UNIX epoch.
	LastUploadTime int64 `json:"lastUploadTime"`
	// The currently-shown listings.
	Listings []Listing `json:"listings"`
	// The currently-shown sales.
	RecentHistory []Sale `json:"recentHistory"`
	// The DC name, if applicable.
	DcName string `json:"dcName"`
	// The region name, if applicable.
	RegionName string `json:"regionName"`
	// The average listing price.
	CurrentAveragePrice float64 `json:"currentAveragePrice"`
	// The average NQ listing price.
	CurrentAveragePriceNq float64 `json:"currentAveragePriceNQ"`
	// The average HQ listing price.
	CurrentAveragePriceHq float64 `json:"currentAveragePriceHQ"`
	// The average number of sales per day, over the past seven days (or the entirety of the shown sales, whichever comes first).
	// This number will tend to be the same for every item, because the number of shown sales is the same and over the same period.
	// This statistic is more useful in historical queries.
	RegularSaleVelocity float64 `json:"regularSaleVelocity"`
	// The average number of NQ sales per day, over the past seven days (or the entirety of the shown sales, whichever comes first).
	// This number will tend to be the same for every item, because the number of shown sales is the same and over the same period.
	// This statistic is more useful in historical queries.
	NqSaleVelocity float64 `json:"nqSaleVelocity"`
	// The average number of HQ sales per day, over the past seven days (or the entirety of the shown sales, whichever comes first).
	// This number will tend to be the same for every item, because the number of shown sales is the same and over the same period.
	// This statistic is more useful in historical queries.
	HqSaleVelocity float64 `json:"hqSaleVelocity"`
	// The average sale price.
	AveragePrice float64 `json:"averagePrice"`
	// The average NQ sale price.
	AveragePriceNq float64 `json:"averagePriceNQ"`
	// The average HQ sale price.
	AveragePriceHq float64 `json:"averagePriceHQ"`
	// The minimum listing price.
	MinPrice int32 `json:"minPrice"`
	// The minimum NQ listing price.
	MinPriceNq int32 `json:"minPriceNQ"`
	// The minimum HQ listing price.
	MinPriceHq int32 `json:"minPriceHQ"`
	// The maximum listing price.
	MaxPrice int32 `json:"maxPrice"`
	// The maximum NQ listing price.
	MaxPriceNq int32 `json:"maxPriceNQ"`
	// The maximum HQ listing price.
	MaxPriceHq int32 `json:"maxPriceHQ"`
	// A map of quantities to listing counts, representing the number of listings of each quantity.
	StackSizeHistogram json.RawMessage `json:"stackSizeHistogram"`
	// A map of quantities to NQ listing counts, representing the number of listings of each quantity.
	StackSizeHistogramNq json.RawMessage `json:"stackSizeHistogramNQ"`
	// A map of quantities to HQ listing counts, representing the number of listings of each quantity.
	StackSizeHistogramHq json.RawMessage `json:"stackSizeHistogramHQ"`
	// The world name, if applicable.
	WorldName string `json:"worldName"`
	// The last upload times in milliseconds since epoch for each world in the response, if this is a DC request.
	WorldUploadTimes json.RawMessage `json:"worldUploadTimes"`
	// The number of listings retrieved for the request. When using the "listings" limit parameter, this may be
	// different from the number of sale entries returned in an API response.
	ListingsCount int32 `json:"listingsCount"`
	// The number of sale entries retrieved for the request. When using the "entries" limit parameter, this may be
	// different from the number of sale entries returned in an API response.
	RecentHistoryCount int32 `json:"recentHistoryCount"`
	// The number of items (not listings) up for sale.
	UnitsForSale int32 `json:"unitsForSale"`
	// The number of items (not sale entries) sold over the retrieved sales.
	UnitsSold int32 `json:"unitsSold"`
	// Whether this item has ever been updated. Useful for newly-released items.
	HasData bool `json:"hasData"`
}

type Listing struct {
	// The item ID.
	ItemId int32 `json:"itemID"`
	// The time that this listing was posted, in seconds since the UNIX epoch.
	LastReviewTime int64 `json:"lastReviewTime"`
	// The price per unit sold.
	PricePerUnit int32 `json:"pricePerUnit"`
	// The stack size sold.
	Quantity int32 `json:"quantity"`
	// The ID of the dye on this item.
	StainId int32 `json:"stainID"`
	// The world name, if applicable.
	WorldName string `json:"worldName"`
	// The world ID, if applicable.
	WorldId int32 `json:"worldID"`
	// The creator's character name.
	CreatorName string `json:"creatorName"`
	// A SHA256 hash of the creator's ID.
	CreatorId string `json:"creatorID"`
	// Whether or not the item is high-quality.
	Hq bool `json:"hq"`
	// Whether or not the item is crafted.
	IsCrafted bool `json:"isCrafted"`
	// The ID of this listing.
	ListingId string `json:"listingID"`
	// The materia on this item.
	Materia []Materia `json:"materia"`
	// Whether or not the item is being sold on a mannequin.
	OnMannequin bool `json:"onMannequin"`
	// The city ID of the retainer. This is a game ID; all possible values can be seen at
	// https://xivapi.com/Town.
	//
	// Limsa Lominsa = 1
	// Gridania = 2
	// Ul'dah = 3
	// Ishgard = 4
	// Kugane = 7
	// Crystarium = 10
	// Old Sharlayan = 12
	RetainerCity int32 `json:"retainerCity"`
	// The retainer's ID.
	RetainerId string `json:"retainerID"`
	// The retainer's name.
	RetainerName string `json:"retainerName"`
	// A SHA256 hash of the seller's ID.
	SellerId string `json:"sellerID"`
	// The total price.
	Total int32 `json:"total"`
	// The Gil sales tax (GST) to be added to the total price during purchase.
	Tax int32 `json:"tax"`
}

type Materia struct {
	// The materia slot.
	SlotId int32 `json:"slotID"`
	// The materia item ID.
	MateriaId int32 `json:"materiaID"`
}

type Sale struct {
	// The item ID.
	ItemId int32 `json:"itemID"`
	// Whether or not the item was high-quality.
	Hq bool `json:"hq"`
	// The price per unit sold.
	PricePerUnit int32 `json:"pricePerUnit"`
	// The stack size sold.
	Quantity int32 `json:"quantity"`
	// The sale time, in seconds since the UNIX epoch.
	Timestamp int64 `json:"timestamp"`
	// Whether or not this was purchased from a mannequin. This may be null.
	OnMannequin bool `json:"onMannequin"`
	// The world name, if applicable.
	WorldName string `json:"worldName"`
	// The world ID, if applicable.
	WorldId int32 `json:"worldID"`
	// The buyer name.
	BuyerName string `json:"buyerName"`
	// The total price.
	Total int32 `json:"total"`
}

type WorldItemRecency struct {
	// The item ID.
	ItemId int32 `json:"itemID"`
	// The last upload time for the item on the listed world.
	LastUploadTime int64 `json:"lastUploadTime"`
	// The world ID.
	// WorldId int32 `json:"worldID"`
	// The world name.
	WorldName string `json:"worldName"`
}

type MultiView[T any] struct {
	// The item IDs that were requested.
	ItemIds []int32 `json:"itemIDs"`
	// The item data that was requested, keyed on the item ID.
	Items map[string]T `json:"items"`
	// The ID of the world requested, if applicable.
	WorldId int32 `json:"worldID"`
	// The name of the DC requested, if applicable.
	DcName string `json:"dcName"`
	// The name of the region requested, if applicable.
	RegionName string `json:"regionName"`
	// A list of IDs that could not be resolved to any item data.
	UnresolvedItems []int32 `json:"unresolvedItems"`
	// The name of the world requested, if applicable.
	WorldName string `json:"worldName"`
}

type DataCenter struct {
	Name   string  `json:"name"`
	Region string  `json:"region"`
	Worlds []int32 `json:"worlds"`
}
