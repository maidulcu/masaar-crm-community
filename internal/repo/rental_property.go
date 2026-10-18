package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type RentalPropertyRepo struct {
	db *pgxpool.Pool
}

func NewRentalPropertyRepo(db *pgxpool.Pool) *RentalPropertyRepo {
	return &RentalPropertyRepo{db: db}
}

func (r *RentalPropertyRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.RentalProperty], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM rental_properties WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count properties: %w", err)
	}

	const q = `
		SELECT id, company_id, name, description, property_type, units_count, area, street_address,
		       building_number, unit_number, city, emirate, postal_code, total_sqft, bedrooms, bathrooms,
		       parking_spaces, amenities, purchase_price, purchase_date, market_value, currency, status,
		       occupancy_status, total_occupied_units, property_deed_url, title_deed_number,
		       municipality_registration, created_at, updated_at, created_by, updated_by
		FROM rental_properties
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list properties: %w", err)
	}
	defer rows.Close()

	var properties []domain.RentalProperty
	for rows.Next() {
		var p domain.RentalProperty
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.PropertyType, &p.UnitsCount, &p.Area, &p.StreetAddress,
			&p.BuildingNumber, &p.UnitNumber, &p.City, &p.Emirate, &p.PostalCode, &p.TotalSqft, &p.Bedrooms, &p.Bathrooms,
			&p.ParkingSpaces, &p.Amenities, &p.PurchasePrice, &p.PurchaseDate, &p.MarketValue, &p.Currency,
			&p.Status, &p.OccupancyStatus, &p.TotalOccupiedUnits, &p.PropertyDeedURL, &p.TitleDeedNumber,
			&p.MunicipalityRegNum, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		properties = append(properties, p)
	}

	return &domain.PaginatedResult[domain.RentalProperty]{
		Data:  properties,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *RentalPropertyRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RentalProperty, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT id, company_id, name, description, property_type, units_count, area, street_address,
		       building_number, unit_number, city, emirate, postal_code, total_sqft, bedrooms, bathrooms,
		       parking_spaces, amenities, purchase_price, purchase_date, market_value, currency, status,
		       occupancy_status, total_occupied_units, property_deed_url, title_deed_number,
		       municipality_registration, created_at, updated_at, created_by, updated_by
		FROM rental_properties WHERE id = $1 AND company_id = $2
	`
	p := &domain.RentalProperty{}
	err = r.db.QueryRow(ctx, q, id, cid).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.PropertyType, &p.UnitsCount, &p.Area, &p.StreetAddress,
		&p.BuildingNumber, &p.UnitNumber, &p.City, &p.Emirate, &p.PostalCode, &p.TotalSqft, &p.Bedrooms, &p.Bathrooms,
		&p.ParkingSpaces, &p.Amenities, &p.PurchasePrice, &p.PurchaseDate, &p.MarketValue, &p.Currency,
		&p.Status, &p.OccupancyStatus, &p.TotalOccupiedUnits, &p.PropertyDeedURL, &p.TitleDeedNumber,
		&p.MunicipalityRegNum, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get property by id: %w", err)
	}
	return p, nil
}

func (r *RentalPropertyRepo) Create(ctx context.Context, p *domain.RentalProperty) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	p.CompanyID = cid // never trust a company id supplied by the client
	const q = `
		INSERT INTO rental_properties (
			id, company_id, name, description, property_type, units_count, area, street_address,
			building_number, unit_number, city, emirate, postal_code, total_sqft, bedrooms, bathrooms,
			parking_spaces, amenities, purchase_price, purchase_date, market_value, currency, status,
			occupancy_status, total_occupied_units, property_deed_url, title_deed_number,
			municipality_registration, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		)
		RETURNING created_at, updated_at
	`
	p.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		p.ID, p.CompanyID, p.Name, p.Description, p.PropertyType, p.UnitsCount, p.Area, p.StreetAddress,
		p.BuildingNumber, p.UnitNumber, p.City, p.Emirate, p.PostalCode, p.TotalSqft, p.Bedrooms, p.Bathrooms,
		p.ParkingSpaces, p.Amenities, p.PurchasePrice, p.PurchaseDate, p.MarketValue, p.Currency,
		p.Status, p.OccupancyStatus, p.TotalOccupiedUnits, p.PropertyDeedURL, p.TitleDeedNumber,
		p.MunicipalityRegNum, p.CreatedBy, p.UpdatedBy,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}

// ErrRentalPropertyNotFound is returned when a property does not exist in the caller's company. It
// wraps pgx.ErrNoRows so handlers answer 404.
var ErrRentalPropertyNotFound = fmt.Errorf("rental property not found: %w", pgx.ErrNoRows)

// ErrPropertyHasWork is returned by Delete when maintenance tasks or inspections still refer to the property.
var ErrPropertyHasWork = errors.New("property has maintenance tasks or inspections")

func (r *RentalPropertyRepo) Update(ctx context.Context, p *domain.RentalProperty) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	const q = `
		UPDATE rental_properties
		SET name=$1, description=$2, property_type=$3, units_count=$4, area=$5, street_address=$6,
		    building_number=$7, unit_number=$8, city=$9, emirate=$10, postal_code=$11, total_sqft=$12,
		    bedrooms=$13, bathrooms=$14, parking_spaces=$15, amenities=$16, market_value=$17, status=$18,
		    occupancy_status=$19, total_occupied_units=$20, property_deed_url=$21, title_deed_number=$22,
		    municipality_registration=$23, updated_by=$24, updated_at=NOW(),
		    currency=$27, purchase_price=$28, purchase_date=$29
		WHERE id=$25 AND company_id=$26
		RETURNING updated_at
	`
	err = r.db.QueryRow(ctx, q,
		p.Name, p.Description, p.PropertyType, p.UnitsCount, p.Area, p.StreetAddress,
		p.BuildingNumber, p.UnitNumber, p.City, p.Emirate, p.PostalCode, p.TotalSqft,
		p.Bedrooms, p.Bathrooms, p.ParkingSpaces, p.Amenities, p.MarketValue, p.Status,
		p.OccupancyStatus, p.TotalOccupiedUnits, p.PropertyDeedURL, p.TitleDeedNumber,
		p.MunicipalityRegNum, p.UpdatedBy, p.ID, cid,
		p.Currency, p.PurchasePrice, p.PurchaseDate,
	).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRentalPropertyNotFound
	}
	return err
}

func (r *RentalPropertyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// Maintenance tasks and inspections have no foreign key to the property, so deleting it left
	// them orphaned (still listed, pointing at nothing).
	var busy bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM maintenance_tasks WHERE property_id = $1 AND company_id = $2 AND deleted_at IS NULL)
		    OR EXISTS (SELECT 1 FROM inspections WHERE property_id = $1 AND company_id = $2)`, id, cid).Scan(&busy); err != nil {
		return err
	}
	if busy {
		return ErrPropertyHasWork
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM rental_properties WHERE id=$1 AND company_id=$2`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRentalPropertyNotFound
	}
	return nil
}

func (r *RentalPropertyRepo) ListByArea(ctx context.Context, companyID uuid.UUID, area string, page, limit int) (*domain.PaginatedResult[domain.RentalProperty], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM rental_properties WHERE company_id = $1 AND area = $2`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID, area).Scan(&total); err != nil {
		return nil, fmt.Errorf("count properties by area: %w", err)
	}

	const q = `
		SELECT id, company_id, name, description, property_type, units_count, area, street_address,
		       building_number, unit_number, city, emirate, postal_code, total_sqft, bedrooms, bathrooms,
		       parking_spaces, amenities, purchase_price, purchase_date, market_value, currency, status,
		       occupancy_status, total_occupied_units, property_deed_url, title_deed_number,
		       municipality_registration, created_at, updated_at, created_by, updated_by
		FROM rental_properties
		WHERE company_id = $1 AND area = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, q, companyID, area, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list properties by area: %w", err)
	}
	defer rows.Close()

	var properties []domain.RentalProperty
	for rows.Next() {
		var p domain.RentalProperty
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.Name, &p.Description, &p.PropertyType, &p.UnitsCount, &p.Area, &p.StreetAddress,
			&p.BuildingNumber, &p.UnitNumber, &p.City, &p.Emirate, &p.PostalCode, &p.TotalSqft, &p.Bedrooms, &p.Bathrooms,
			&p.ParkingSpaces, &p.Amenities, &p.PurchasePrice, &p.PurchaseDate, &p.MarketValue, &p.Currency,
			&p.Status, &p.OccupancyStatus, &p.TotalOccupiedUnits, &p.PropertyDeedURL, &p.TitleDeedNumber,
			&p.MunicipalityRegNum, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		properties = append(properties, p)
	}

	return &domain.PaginatedResult[domain.RentalProperty]{
		Data:  properties,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
