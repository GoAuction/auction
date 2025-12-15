package grpc

import (
	"auction/app"
	itemv1 "auction/proto/gen"
	"context"
	"database/sql"
	"errors"

	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ItemServiceServer struct {
	itemv1.UnimplementedItemServiceServer
	repository app.Repository
}

func NewItemServiceServer(repository app.Repository) *ItemServiceServer {
	return &ItemServiceServer{
		repository: repository,
	}
}

func (s *ItemServiceServer) GetItemForBid(ctx context.Context, req *itemv1.GetItemForBidRequest) (*itemv1.GetItemForBidResponse, error) {
	if req.ItemId == "" {
		return nil, status.Error(codes.InvalidArgument, "item_id is required")
	}

	item, err := s.repository.GetItem(ctx, req.ItemId)
	if err != nil {
		return nil, s.mapError(err)
	}

	return &itemv1.GetItemForBidResponse{
		Id:           item.ID,
		SellerId:     item.SellerID,
		Status:       item.Status,
		StartDate:    timestamppb.New(item.StartDate),
		EndDate:      timestamppb.New(item.EndDate),
		StartPrice:   item.StartPrice.String(),
		CurrentPrice: item.CurrentPrice.String(),
		BidIncrement: decimalToString(item.BidIncrement),
		ReservePrice: decimalToString(item.ReservePrice),
		BuyoutPrice:  decimalToString(item.BuyoutPrice),
		EndPrice:     decimalToString(item.EndPrice),
		CreatedAt:    timestamppb.New(item.CreatedAt),
		UpdatedAt:    timestamppb.New(item.UpdatedAt),
	}, nil
}

func (s *ItemServiceServer) mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.NotFound, "item not found")
	}
	return status.Error(codes.Internal, "internal error")
}

func decimalToString(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}
