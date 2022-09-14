package models

type PrescriptionBody struct {
	Id            string `json:"_id,omitempty"`
	Code          string `json:"code"`
	PatientEmail  string `json:"patientEmail"`
	PatientId     string `json:"patientId" bson:"patientId"`
	PrescriptorId string `json:"prescriptorId"`
	Product       string `json:"product"`
}
