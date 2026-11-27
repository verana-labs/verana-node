package types

import "time"

// Record returns the stored form of xr.
func (xr *ExchangeRate) Record() ExchangeRateRecord {
	return ExchangeRateRecord{
		BaseAssetType:    xr.BaseAssetType,
		BaseAsset:        xr.BaseAsset,
		QuoteAssetType:   xr.QuoteAssetType,
		QuoteAsset:       xr.QuoteAsset,
		Rate:             xr.Rate,
		RateScale:        xr.RateScale,
		ValidityDuration: uint32(xr.ValidityDuration / time.Second),
		Updated:          xr.Updated.Unix(),
		Expires:          xr.Expires.Unix(),
		State:            xr.State,
	}
}

// ExchangeRate returns the entry stored under id.
func (r *ExchangeRateRecord) ExchangeRate(id uint64) ExchangeRate {
	return ExchangeRate{
		Id:               id,
		BaseAssetType:    r.BaseAssetType,
		BaseAsset:        r.BaseAsset,
		QuoteAssetType:   r.QuoteAssetType,
		QuoteAsset:       r.QuoteAsset,
		Rate:             r.Rate,
		RateScale:        r.RateScale,
		ValidityDuration: time.Duration(r.ValidityDuration) * time.Second,
		Updated:          time.Unix(r.Updated, 0).UTC(),
		Expires:          time.Unix(r.Expires, 0).UTC(),
		State:            r.State,
	}
}

// Record returns the stored form of a.
func (a *ExchangeRateAuthorization) Record() ExchangeRateAuthorizationRecord {
	return ExchangeRateAuthorizationRecord{
		Expiration:      a.Expiration.Unix(),
		MinInterval:     uint32(a.MinInterval / time.Second),
		MaxDeviationBps: a.MaxDeviationBps,
	}
}

// Authorization returns the authorization stored under (xrID, feeder).
func (r *ExchangeRateAuthorizationRecord) Authorization(xrID uint64, feeder string) ExchangeRateAuthorization {
	return ExchangeRateAuthorization{
		XrId:            xrID,
		Feeder:          feeder,
		Expiration:      time.Unix(r.Expiration, 0).UTC(),
		MinInterval:     time.Duration(r.MinInterval) * time.Second,
		MaxDeviationBps: r.MaxDeviationBps,
	}
}
