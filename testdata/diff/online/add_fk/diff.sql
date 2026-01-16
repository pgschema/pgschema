ALTER TABLE z_companies
ADD CONSTRAINT z_companies_company_id_name_key UNIQUE (company_id, company_name);

ALTER TABLE a_employees
ADD CONSTRAINT a_employees_company_fkey FOREIGN KEY (tenant_id, company_id) REFERENCES z_companies (tenant_id, company_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

ALTER TABLE a_employees
ADD CONSTRAINT a_employees_company_name_fkey FOREIGN KEY (company_id, company_name) REFERENCES z_companies (company_id, company_name) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
