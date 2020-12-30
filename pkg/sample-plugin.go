package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"

	_ "github.com/lib/pq"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var db *sql.DB
var err error

func newDatasource() datasource.ServeOpts {

	connStr := "postgres://postgres@localhost/postgres"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &SampleDatasource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler: ds,
	}
}

type SampleDatasource struct {
	im instancemgmt.InstanceManager
}

func (td *SampleDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData", "request", req)

	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		res := td.query(ctx, q)
		response.Responses[q.RefID] = res
	}
	return response, nil
}

type queryModel struct {
	Format string `json:"format"`
}
type sandbox struct {
	id    string
	name  string
	phone string
}

func (td *SampleDatasource) query(ctx context.Context, query backend.DataQuery) backend.DataResponse {
	var qm queryModel

	response := backend.DataResponse{}

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	if qm.Format == "" {
		log.DefaultLogger.Warn("format is empty. defaulting to time series")
	}
	frame := data.NewFrame("response")

	rows, err := db.Query("SELECT * FROM newtable")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	snbs := make([]sandbox, 0)

	for rows.Next() {
		snb := sandbox{}
		err := rows.Scan(&snb.id, &snb.name, &snb.phone)
		if err != nil {
			panic(err)
		}
		snbs = append(snbs, snb)
	}

	if err = rows.Err(); err != nil {
		panic(err)
	}

	values := reflect.ValueOf(snbs[0])
	types := values.Type()
	for i := 0; i < values.NumField(); i++ {
		key := types.Field(i).Name
		items := []string{}
		for _, snb := range snbs {
			r := reflect.ValueOf(snb)
			f := reflect.Indirect(r).FieldByName(key)
			log.DefaultLogger.Warn("11111111111 " + f.String())

			items = append(items, f.String())
		}
		frame.Fields = append(frame.Fields, data.NewField(key, nil, items))
	}

	response.Frames = append(response.Frames, frame)

	return response
}

type instanceSettings struct {
	httpClient *http.Client
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &instanceSettings{
		httpClient: &http.Client{},
	}, nil
}

func (s *instanceSettings) Dispose() {
}
