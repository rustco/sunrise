package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sunriselayer/sunrise/x/tokenconverter/types"
)

func (k Keeper) Convert(ctx context.Context, amount math.Int, address sdk.AccAddress) error {
	params := k.GetParams(ctx)

	bondToken := sdk.NewCoin(params.BondDenom, amount)
	if err := bondToken.Validate(); err != nil {
		return err
	}
	feeToken := sdk.NewCoin(params.FeeDenom, amount)
	if err := feeToken.Validate(); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, address, types.ModuleName, sdk.NewCoins(bondToken)); err != nil {
		return err
	}

	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(bondToken)); err != nil {
		return err
	}

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(feeToken)); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, address, sdk.NewCoins(feeToken)); err != nil {
		return err
	}

	if err := sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(&types.EventConvert{
		Address: address.String(),
		Amount:  amount.String(),
	}); err != nil {
		return err
	}

	return nil
}
