-- +goose Up
-- SQL in this section is executed when the migration is applied.

-- Expense Categories table
CREATE TABLE expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    category_name VARCHAR(100) NOT NULL,
    category_type VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, category_name)
);

-- Expenses table
CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    category_id UUID NOT NULL REFERENCES expense_categories(id),
    property_id UUID REFERENCES rental_properties(id),
    tenant_id UUID REFERENCES tenants(id),
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'AED',
    expense_date DATE NOT NULL,
    description TEXT,
    vendor_name VARCHAR(150),
    vendor_contact VARCHAR(255),
    payment_method VARCHAR(50),
    payment_status VARCHAR(50),
    receipt_url VARCHAR(500),
    notes TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Expense Approvals table
CREATE TABLE expense_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id UUID NOT NULL UNIQUE REFERENCES expenses(id),
    approval_status VARCHAR(50) NOT NULL,
    approved_by UUID REFERENCES users(id),
    approval_comments TEXT,
    approval_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_expenses_company_id ON expenses(company_id);
CREATE INDEX idx_expenses_property_id ON expenses(property_id);
CREATE INDEX idx_expenses_expense_date ON expenses(expense_date);
CREATE INDEX idx_expenses_category_id ON expenses(category_id);
CREATE INDEX idx_expense_approvals_status ON expense_approvals(approval_status);
CREATE INDEX idx_expense_categories_company ON expense_categories(company_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP INDEX IF EXISTS idx_expense_categories_company;
DROP INDEX IF EXISTS idx_expense_approvals_status;
DROP INDEX IF EXISTS idx_expenses_category_id;
DROP INDEX IF EXISTS idx_expenses_expense_date;
DROP INDEX IF EXISTS idx_expenses_property_id;
DROP INDEX IF EXISTS idx_expenses_company_id;
DROP TABLE IF EXISTS expense_approvals;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;
