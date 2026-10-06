package seeder

import (
	"context"

	db "github.com/MamangRust/microservice-ecommerce-grpc-merchant_policy/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/logger"

	"go.uber.org/zap"
)

type merchantPolicySeeder struct {
	db     *db.Queries
	ctx    context.Context
	logger logger.LoggerInterface
}

func NewMerchantPolicySeeder(db *db.Queries, ctx context.Context, logger logger.LoggerInterface) *merchantPolicySeeder {
	return &merchantPolicySeeder{
		db:     db,
		ctx:    ctx,
		logger: logger,
	}
}

func (m *merchantPolicySeeder) Seed() error {
	// Idempotency: skip when policies already exist.
	existing, err := m.db.GetMerchantPolicies(m.ctx, db.GetMerchantPoliciesParams{
		Column1: "",
		Limit:   1,
		Offset:  0,
	})
	if err == nil && len(existing) > 0 {
		m.logger.Debug("merchant policies already seeded, skipping")
		return nil
	}

	policies := []db.CreateMerchantPolicyParams{
		{
			MerchantID:  1,
			PolicyType:  "Shipping",
			Title:       "Free Shipping on Orders Above $50",
			Description: "Kami menyediakan pengiriman gratis untuk pembelian di atas $50 di seluruh wilayah Indonesia.",
		},
		{
			MerchantID:  2,
			PolicyType:  "Refund",
			Title:       "30-Day Refund Policy",
			Description: "Pengembalian dana dapat dilakukan dalam waktu 30 hari setelah pembelian dengan syarat dan ketentuan yang berlaku.",
		},
		{
			MerchantID:  3,
			PolicyType:  "Return",
			Title:       "Return Within 7 Days",
			Description: "Produk dapat dikembalikan dalam waktu 7 hari jika tidak sesuai dengan deskripsi atau rusak saat diterima.",
		},
		{
			MerchantID:  4,
			PolicyType:  "Shipping",
			Title:       "Next-Day Delivery in Major Cities",
			Description: "Pengiriman pada hari berikutnya untuk alamat yang berada di kota-kota besar seperti Jakarta, Surabaya, dan Bandung.",
		},
		{
			MerchantID:  5,
			PolicyType:  "Privacy",
			Title:       "Data Privacy Policy",
			Description: "Kami menjaga privasi data pribadi pelanggan dengan sangat hati-hati dan tidak akan membagikan informasi kepada pihak ketiga.",
		},
		{
			MerchantID:  6,
			PolicyType:  "Shipping",
			Title:       "International Shipping Available",
			Description: "Kami menawarkan pengiriman internasional ke berbagai negara di Asia, Eropa, dan Amerika.",
		},
		{
			MerchantID:  7,
			PolicyType:  "Return",
			Title:       "No Returns on Sale Items",
			Description: "Barang yang dibeli dengan harga diskon atau promo tidak dapat dikembalikan.",
		},
		{
			MerchantID:  8,
			PolicyType:  "Refund",
			Title:       "Refund for Defective Products",
			Description: "Kami memberikan pengembalian dana penuh untuk produk yang rusak atau cacat pabrik.",
		},
	}

	for _, policy := range policies {
		if _, err := m.db.CreateMerchantPolicy(m.ctx, policy); err != nil {
			m.logger.Error("failed to seed merchant policy", zap.Error(err))
			return err
		}
	}

	m.logger.Info("merchant policy seeded successfully")

	return nil
}
