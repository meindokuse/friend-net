module github.com/meindokuse/cloud-drive/analytic-service

go 1.25.0

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.30.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/meindokuse/cloud-drive/common v0.0.0-00010101000000-000000000000
	github.com/segmentio/kafka-go v0.4.51
)

replace github.com/meindokuse/cloud-drive/common => ../common
