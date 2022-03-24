package output

import (
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/output"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/old/output/writer"
)

func init() {
	Constructors[TypeAzureBlobStorage] = TypeSpec{
		constructor: fromSimpleConstructor(NewAzureBlobStorage),
		Status:      docs.StatusBeta,
		Version:     "3.36.0",
		Summary: `
Sends message parts as objects to an Azure Blob Storage Account container. Each
object is uploaded with the filename specified with the ` + "`container`" + `
field.`,
		Description: `
Only one authentication method is required, ` + "`storage_connection_string`" + ` or ` + "`storage_account` and `storage_access_key`" + `. If both are set then the ` + "`storage_connection_string`" + ` is given priority.

In order to have a different path for each object you should use function
interpolations described [here](/docs/configuration/interpolation#bloblang-queries), which are
calculated per message of a batch.`,
		Async: true,
		Config: docs.FieldComponent().WithChildren(
			docs.FieldString(
				"storage_account",
				"The storage account to upload messages to. This field is ignored if `storage_connection_string` is set.",
			),
			docs.FieldString(
				"storage_access_key",
				"The storage account access key. This field is ignored if `storage_connection_string` is set.",
			),
			docs.FieldString(
				"storage_sas_token",
				"The storage account SAS token. This field is ignored if `storage_connection_string` or `storage_access_key` / `storage_sas_token` are set.",
			).AtVersion("3.38.0"),
			docs.FieldString(
				"storage_connection_string",
				"A storage account connection string. This field is required if `storage_account` and `storage_access_key` are not set.",
			),
			docs.FieldString("public_access_level", `The container's public access level. The default value is `+"`PRIVATE`"+`.`).HasOptions(
				"PRIVATE", "BLOB", "CONTAINER",
			).Advanced(),
			docs.FieldString(
				"container", "The container for uploading the messages to.",
				`messages-${!timestamp("2006")}`,
			).IsInterpolated(),
			docs.FieldString(
				"path", "The path of each message to upload.",
				`${!count("files")}-${!timestamp_unix_nano()}.json`,
				`${!meta("kafka_key")}.json`,
				`${!json("doc.namespace")}/${!json("doc.id")}.json`,
			).IsInterpolated(),
			docs.FieldString("blob_type", "Block and Append blobs are comprised of blocks, and each blob can support up to 50,000 blocks. The default value is `+\"`BLOCK`\"+`.`").HasOptions(
				"BLOCK", "APPEND",
			).IsInterpolated().Advanced(),
			docs.FieldInt("max_in_flight", "The maximum number of messages to have in flight at a given time. Increase this to improve throughput."),
		),
		Categories: []string{
			"Services",
			"Azure",
		},
	}
}

//------------------------------------------------------------------------------

// NewAzureBlobStorage creates a new AzureBlobStorage output type.
func NewAzureBlobStorage(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (output.Streamed, error) {
	blobStorage, err := writer.NewAzureBlobStorageV2(mgr, conf.AzureBlobStorage, log, stats)
	if err != nil {
		return nil, err
	}
	a, err := NewAsyncWriter(
		TypeAzureBlobStorage, conf.AzureBlobStorage.MaxInFlight, blobStorage, log, stats,
	)
	if err != nil {
		return nil, err
	}
	return OnlySinglePayloads(a), nil
}
