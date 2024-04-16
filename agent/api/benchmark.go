package api

type benchmark struct {
	BeginningTime            int64 `json:"beginning_time,omitempty"`
	MeasurementBeginningTime int64 `json:"measurement_beginning_time,omitempty"`
	MeasurementEndingTime    int64 `json:"measurement_ending_time,omitempty"`
	EndingTime               int64 `json:"ending_time,omitempty"`
}
