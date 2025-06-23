package operator

import (
	"encoding/json"
	"fmt"

	aptos "github.com/aptos-labs/aptos-go-sdk"
)

// TODO
const FAContract = "0x24f4512994b08ea6f6914a776670d0e06541b2e72ffcbf32a279be4c8eb4e845"

func GetMetadata(
	client *aptos.Client,
) Metadata {
	contractAcc := aptos.AccountAddress{}
	err := contractAcc.ParseStringRelaxed(FAContract)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	var noTypeTags []aptos.TypeTag
	viewResponse, err := client.View(&aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contractAcc,
			Name:    "fungible_asset",
		},
		Function: "get_metadata",
		ArgTypes: noTypeTags,
		Args:     [][]byte{},
	})
	if err != nil {
		panic("Failed to view fa address:" + err.Error())
	}
	metadataMap := viewResponse[0].(map[string]interface{})
	metadataBz, err := json.Marshal(metadataMap)
	if err != nil {
		panic("Failed to marshal metadata to json:" + err.Error())
	}

	var metadataStr MetadataStr
	err = json.Unmarshal(metadataBz, &metadataStr)
	if err != nil {
		panic("Failed to unmarshal metadata from json:" + err.Error())
	}
	metadataAcc := aptos.AccountAddress{}
	err = metadataAcc.ParseStringRelaxed(metadataStr.Inner)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	fmt.Println("metadata: ", metadataAcc.String())
	return Metadata{
		Inner: metadataAcc,
	}
}

func FAMetdataClient(
	client *aptos.Client,
) *aptos.FungibleAssetClient {
	contractAcc := aptos.AccountAddress{}
	err := contractAcc.ParseStringRelaxed(FAContract)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	var noTypeTags []aptos.TypeTag
	viewResponse, err := client.View(&aptos.ViewPayload{
		Module: aptos.ModuleId{
			Address: contractAcc,
			Name:    "fungible_asset",
		},
		Function: "get_metadata",
		ArgTypes: noTypeTags,
		Args:     [][]byte{},
	})
	if err != nil {
		panic("Failed to view fa address:" + err.Error())
	}

	metadataMap := viewResponse[0].(map[string]interface{})
	metadataBz, err := json.Marshal(metadataMap)
	if err != nil {
		panic("Failed to marshal metadata to json:" + err.Error())
	}

	var metadataStr MetadataStr
	err = json.Unmarshal(metadataBz, &metadataStr)
	if err != nil {
		panic("Failed to unmarshal metadata from json:" + err.Error())
	}
	metadataAcc := aptos.AccountAddress{}
	err = metadataAcc.ParseStringRelaxed(metadataStr.Inner)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	faClient, err := aptos.NewFungibleAssetClient(client, &metadataAcc)
	if err != nil {
		panic("Failed to create fa client:" + err.Error())
	}
	return faClient
}
