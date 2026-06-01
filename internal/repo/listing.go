package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type ListingRepo struct {
	db *pgxpool.Pool
}

func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

func (r *ListingRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.Listing], error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM listings WHERE company_id = $1`, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count listings: %w", err)
	}

	rows, err := r.db.Query(ctx, listQuery, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list listings: %w", err)
	}
	defer rows.Close()

	listings, err := scanListings(rows)
	if err != nil {
		return nil, err
	}

	return &domain.PaginatedResult[domain.Listing]{
		Data:  listings,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *ListingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Listing, error) {
	rows, err := r.db.Query(ctx, listQuery+` AND l.id = $2`, id)
	if err != nil {
		return nil, fmt.Errorf("get listing: %w", err)
	}
	defer rows.Close()

	listings, err := scanListings(rows)
	if err != nil {
		return nil, err
	}
	if len(listings) == 0 {
		return nil, fmt.Errorf("listing not found")
	}
	return &listings[0], nil
}

func (r *ListingRepo) Create(ctx context.Context, l *domain.Listing) error {
	l.ID = uuid.New()
	_, err := r.db.Exec(ctx, `
		INSERT INTO listings (
			id, company_id, title, description, property_type, listing_type,
			price, currency, rent_period,
			area, community, subcommunity, city, emirate, latitude, longitude,
			bedrooms, bathrooms, total_sqft, plot_sqft, parking_spaces, furnishing, amenities, year_built,
			cover_image_url, image_urls, virtual_tour_url, video_url,
			status, featured, reference_number, available_from,
			assigned_to, owner_name, owner_phone, owner_email,
			portal_sync_status, created_by, updated_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,
			$33,$34,$35,$36,$37,$38,$39
		)
	`, l.ID, l.CompanyID, l.Title, l.Description, l.PropertyType, l.ListingType,
		l.Price, l.Currency, l.RentPeriod,
		l.Area, l.Community, l.Subcommunity, l.City, l.Emirate, l.Latitude, l.Longitude,
		l.Bedrooms, l.Bathrooms, l.TotalSqft, l.PlotSqft, l.ParkingSpaces, l.Furnishing, l.Amenities, l.YearBuilt,
		l.CoverImageURL, l.ImageURLs, l.VirtualTourURL, l.VideoURL,
		l.Status, l.Featured, l.ReferenceNumber, l.AvailableFrom,
		l.AssignedTo, l.OwnerName, l.OwnerPhone, l.OwnerEmail,
		l.PortalSyncStatus, l.CreatedBy, l.UpdatedBy,
	)
	return err
}

func (r *ListingRepo) Update(ctx context.Context, l *domain.Listing) error {
	_, err := r.db.Exec(ctx, `
		UPDATE listings SET
			title=$1, description=$2, property_type=$3, listing_type=$4,
			price=$5, currency=$6, rent_period=$7,
			area=$8, community=$9, subcommunity=$10, city=$11, emirate=$12, latitude=$13, longitude=$14,
			bedrooms=$15, bathrooms=$16, total_sqft=$17, plot_sqft=$18, parking_spaces=$19, furnishing=$20, amenities=$21, year_built=$22,
			cover_image_url=$23, image_urls=$24, virtual_tour_url=$25, video_url=$26,
			status=$27, featured=$28, reference_number=$29, available_from=$30,
			assigned_to=$31, owner_name=$32, owner_phone=$33, owner_email=$34,
			updated_by=$35, updated_at=NOW()
		WHERE id=$36
	`, l.Title, l.Description, l.PropertyType, l.ListingType,
		l.Price, l.Currency, l.RentPeriod,
		l.Area, l.Community, l.Subcommunity, l.City, l.Emirate, l.Latitude, l.Longitude,
		l.Bedrooms, l.Bathrooms, l.TotalSqft, l.PlotSqft, l.ParkingSpaces, l.Furnishing, l.Amenities, l.YearBuilt,
		l.CoverImageURL, l.ImageURLs, l.VirtualTourURL, l.VideoURL,
		l.Status, l.Featured, l.ReferenceNumber, l.AvailableFrom,
		l.AssignedTo, l.OwnerName, l.OwnerPhone, l.OwnerEmail,
		l.UpdatedBy, l.ID,
	)
	return err
}

func (r *ListingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ListingStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE listings SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	return err
}

func (r *ListingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM listings WHERE id=$1`, id)
	return err
}

const listQuery = `
	SELECT l.id, l.company_id,
	       l.title, l.description, l.property_type, l.listing_type,
	       l.price, l.currency, l.rent_period,
	       l.area, l.community, l.subcommunity, l.city, l.emirate, l.latitude, l.longitude,
	       l.bedrooms, l.bathrooms, l.total_sqft, l.plot_sqft, l.parking_spaces, l.furnishing, l.amenities, l.year_built,
	       l.cover_image_url, l.image_urls, l.virtual_tour_url, l.video_url,
	       l.status, l.featured, l.reference_number, l.available_from,
	       l.assigned_to, l.owner_name, l.owner_phone, l.owner_email,
	       l.portal_sync_status,
	       l.published_at, l.created_at, l.updated_at, l.created_by, l.updated_by
	FROM listings l
	WHERE l.company_id = $1
	ORDER BY l.featured DESC, l.created_at DESC
	LIMIT $2 OFFSET $3
`

func scanListings(rows pgx.Rows) ([]domain.Listing, error) {
	var listings []domain.Listing
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(
			&l.ID, &l.CompanyID,
			&l.Title, &l.Description, &l.PropertyType, &l.ListingType,
			&l.Price, &l.Currency, &l.RentPeriod,
			&l.Area, &l.Community, &l.Subcommunity, &l.City, &l.Emirate, &l.Latitude, &l.Longitude,
			&l.Bedrooms, &l.Bathrooms, &l.TotalSqft, &l.PlotSqft, &l.ParkingSpaces, &l.Furnishing, &l.Amenities, &l.YearBuilt,
			&l.CoverImageURL, &l.ImageURLs, &l.VirtualTourURL, &l.VideoURL,
			&l.Status, &l.Featured, &l.ReferenceNumber, &l.AvailableFrom,
			&l.AssignedTo, &l.OwnerName, &l.OwnerPhone, &l.OwnerEmail,
			&l.PortalSyncStatus,
			&l.PublishedAt, &l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan listing: %w", err)
		}
		listings = append(listings, l)
	}
	return listings, nil
}
