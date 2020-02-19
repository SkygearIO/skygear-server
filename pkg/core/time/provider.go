package time

import "time"

type Provider interface {
	NowUTC() time.Time
	Now() time.Time
}

type ProviderImpl struct{}

func NewProvider() ProviderImpl {
	return ProviderImpl{}
}

func (provider ProviderImpl) NowUTC() time.Time {
	return time.Now().UTC()
}

func (provider ProviderImpl) Now() time.Time {
	return time.Now()
}
