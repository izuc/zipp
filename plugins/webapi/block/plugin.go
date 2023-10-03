package block

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/izuc/zipp.foundation/core/stringify"
	"github.com/labstack/echo"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/mesh_old/payload"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/markers"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// region Plugin ///////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	// Plugin holds the singleton instance of the plugin.
	Plugin *node.Plugin

	deps = new(dependencies)
)

type dependencies struct {
	dig.In

	Server *echo.Echo
	Mesh *mesh_old.Mesh
}

func init() {
	Plugin = node.NewPlugin("WebAPIBlockEndpoint", deps, node.Enabled, configure)
}

func configure(_ *node.Plugin) {
	deps.Server.GET("blocks/:blockID", GetBlock)
	deps.Server.GET("blocks/:blockID/metadata", GetBlockMetadata)
	deps.Server.POST("blocks/payload", PostPayload)

	deps.Server.GET("blocks/sequences/:sequenceID", GetSequence)
	deps.Server.GET("blocks/sequences/:sequenceID/markerindexconflictidmapping", GetMarkerIndexConflictIDMapping)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region GetSequence //////////////////////////////////////////////////////////////////////////////////////////////////

// GetSequence is the handler for the /blocks/sequences/:sequenceID endpoint.
func GetSequence(c echo.Context) (err error) {
	sequenceID, err := sequenceIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	if deps.Mesh.Booker.MarkersManager.Sequence(sequenceID).Consume(func(sequence *markers.Sequence) {
		blockWithLastMarker := deps.Mesh.Booker.MarkersManager.BlockID(markers.NewMarker(sequenceID, sequence.HighestIndex()))
		err = c.String(http.StatusOK, stringify.Struct("Sequence",
			stringify.NewStructField("ID", sequence.ID()),
			stringify.NewStructField("LowestIndex", sequence.LowestIndex()),
			stringify.NewStructField("HighestIndex", sequence.HighestIndex()),
			stringify.NewStructField("BlockWithLastMarker", blockWithLastMarker),
		))
	}) {
		return
	}

	return c.JSON(http.StatusNotFound, jsonmodels.NewErrorResponse(fmt.Errorf("failed to load Sequence with %s", sequenceID)))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region GetMarkerIndexConflictIDMapping ////////////////////////////////////////////////////////////////////////////////

// GetMarkerIndexConflictIDMapping is the handler for the /blocks/sequences/:sequenceID/markerindexconflictidmapping endpoint.
func GetMarkerIndexConflictIDMapping(c echo.Context) (err error) {
	sequenceID, err := sequenceIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	if deps.Mesh.Storage.MarkerIndexConflictIDMapping(sequenceID).Consume(func(markerIndexConflictIDMapping *mesh_old.MarkerIndexConflictIDMapping) {
		err = c.String(http.StatusOK, markerIndexConflictIDMapping.String())
	}) {
		return
	}

	return c.JSON(http.StatusNotFound, jsonmodels.NewErrorResponse(fmt.Errorf("failed to load MarkerIndexConflictIDMapping of %s", sequenceID)))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region GetBlock ///////////////////////////////////////////////////////////////////////////////////////////////////

// GetBlock is the handler for the /blocks/:blockID endpoint.
func GetBlock(c echo.Context) (err error) {
	blockID, err := blockIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	if deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
		var payloadBytes []byte
		payloadBytes, err = block.Payload().Bytes()

		ecRecord := epoch.NewECRecord(block.ECRecordEI())
		ecRecord.SetECR(block.ECR())
		ecRecord.SetPrevEC(block.PrevEC())

		err = c.JSON(http.StatusOK, jsonmodels.Block{
			ID:                  block.ID().Base58(),
			StrongParents:       block.ParentsByType(mesh_old.StrongParentType).Base58(),
			WeakParents:         block.ParentsByType(mesh_old.WeakParentType).Base58(),
			ShallowLikeParents:  block.ParentsByType(mesh_old.ShallowLikeParentType).Base58(),
			StrongChildren:      deps.Mesh.Utils.ApprovingBlockIDs(block.ID(), mesh_old.StrongChild).Base58(),
			WeakChildren:        deps.Mesh.Utils.ApprovingBlockIDs(block.ID(), mesh_old.WeakChild).Base58(),
			ShallowLikeChildren: deps.Mesh.Utils.ApprovingBlockIDs(block.ID(), mesh_old.ShallowLikeChild).Base58(),
			IssuerPublicKey:     block.IssuerPublicKey().String(),
			IssuingTime:         block.IssuingTime().Unix(),
			SequenceNumber:      block.SequenceNumber(),
			PayloadType:         block.Payload().Type().String(),
			TransactionID: func() string {
				if block.Payload().Type() == devnetvm.TransactionType {
					return block.Payload().(*devnetvm.Transaction).ID().Base58()
				}

				return ""
			}(),
			EC:                   ecRecord.ComputeEC().Base58(),
			EI:                   uint64(block.ECRecordEI()),
			ECR:                  block.ECR().Base58(),
			PrevEC:               block.PrevEC().Base58(),
			Payload:              payloadBytes,
			Signature:            block.Signature().String(),
			LatestConfirmedEpoch: uint64(block.LatestConfirmedEpoch()),
		})
	}) {
		return
	}

	return c.JSON(http.StatusNotFound, jsonmodels.NewErrorResponse(fmt.Errorf("failed to load Block with %s", blockID)))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region GetBlockMetadata ///////////////////////////////////////////////////////////////////////////////////////////

// GetBlockMetadata is the handler for the /blocks/:blockID/metadata endpoint.
func GetBlockMetadata(c echo.Context) (err error) {
	blockID, err := blockIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	if deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
		err = c.JSON(http.StatusOK, NewBlockMetadata(blockMetadata))
	}) {
		return
	}

	return c.JSON(http.StatusNotFound, jsonmodels.NewErrorResponse(fmt.Errorf("failed to load BlockMetadata with %s", blockID)))
}

// NewBlockMetadata returns BlockMetadata from the given mesh_old.BlockMetadata.
func NewBlockMetadata(metadata *mesh_old.BlockMetadata) jsonmodels.BlockMetadata {
	conflictIDs, _ := deps.Mesh.Booker.BlockConflictIDs(metadata.ID())

	return jsonmodels.BlockMetadata{
		ID:                    metadata.ID().Base58(),
		ReceivedTime:          metadata.ReceivedTime().Unix(),
		Solid:                 metadata.IsSolid(),
		SolidificationTime:    metadata.SolidificationTime().Unix(),
		StructureDetails:      jsonmodels.NewStructureDetails(metadata.StructureDetails()),
		ConflictIDs:           lo.Map(conflictIDs.Slice(), utxo.TransactionID.Base58),
		AddedConflictIDs:      lo.Map(metadata.AddedConflictIDs().Slice(), utxo.TransactionID.Base58),
		SubtractedConflictIDs: lo.Map(metadata.SubtractedConflictIDs().Slice(), utxo.TransactionID.Base58),
		Scheduled:             metadata.Scheduled(),
		ScheduledTime:         metadata.ScheduledTime().Unix(),
		Booked:                metadata.IsBooked(),
		BookedTime:            metadata.BookedTime().Unix(),
		ObjectivelyInvalid:    metadata.IsObjectivelyInvalid(),
		SubjectivelyInvalid:   metadata.IsSubjectivelyInvalid(),
		ConfirmationState:     metadata.ConfirmationState(),
		ConfirmationStateTime: metadata.ConfirmationStateTime().Unix(),
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region PostPayload //////////////////////////////////////////////////////////////////////////////////////////////////

// PostPayload is the handler for the /blocks/payload endpoint.
func PostPayload(c echo.Context) error {
	var request jsonmodels.PostPayloadRequest
	if err := c.Bind(&request); err != nil {
		Plugin.LogInfo(err.Error())
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	parsedPayload, _, err := payload.FromBytes(request.Payload)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	blk, err := deps.Mesh.IssuePayload(parsedPayload)
	if err != nil {
		return c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
	}

	return c.JSON(http.StatusOK, jsonmodels.NewPostPayloadResponse(blk))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region blockIDFromContext /////////////////////////////////////////////////////////////////////////////////////////

// blockIDFromContext determines the BlockID from the blockID parameter in an echo.Context. It expects it to
// either be a base58 encoded string or the builtin alias EmptyBlockID.
func blockIDFromContext(c echo.Context) (blockID mesh_old.BlockID, err error) {
	switch blockIDString := c.Param("blockID"); blockIDString {
	case "EmptyBlockID":
		blockID = mesh_old.EmptyBlockID
	default:
		err = blockID.FromBase58(blockIDString)
	}

	return
}

// sequenceIDFromContext determines the sequenceID from the sequenceID parameter in an echo.Context.
func sequenceIDFromContext(c echo.Context) (id markers.SequenceID, err error) {
	sequenceIDInt, err := strconv.Atoi(c.Param("sequenceID"))
	if err != nil {
		return
	}

	return markers.SequenceID(sequenceIDInt), nil
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
