ALTER TABLE employees
ADD CONSTRAINT employees_company_fkey FOREIGN KEY (tenant_id, company_id) REFERENCES companies (tenant_id, company_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
