#!/usr/bin/env python3
"""
PostgreSQL CRUD integration test via adb-link HTTP API.

This script tests:
1. Create database "test" on pg-local datasource
2. Create tables (employees, departments)
3. INSERT data
4. SELECT data
5. UPDATE data
6. DELETE data
7. Cleanup (drop tables and database)

Usage:
    python3 tests/test_pg_crud.py
"""

import sys
import json
import requests

BASE_URL = "http://localhost:8000"
API_KEY = "6b023a18586fc9dbdfb887ce04e23faa"
DATASOURCE = "pg-local"
TEST_DB = "test"

HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json",
}


def execute_sql(sql: str, database: str = "postgres", datasource: str = DATASOURCE) -> dict:
    """Execute SQL via the adb-link API and return the response dict."""
    payload = {
        "datasource_name": datasource,
        "database": database,
        "sql": sql,
        "limit": 1000,
        "timeout_seconds": 30,
    }
    resp = requests.post(f"{BASE_URL}/api/query/execute", headers=HEADERS, json=payload)
    resp.raise_for_status()
    data = resp.json()
    return data


def assert_success(result: dict, context: str):
    """Assert the API response indicates success."""
    if not result.get("success"):
        print(f"  FAIL [{context}]: {result.get('error', 'unknown error')}")
        sys.exit(1)
    print(f"  OK   [{context}]")


def step(msg: str):
    """Print a step header."""
    print(f"\n{'='*60}")
    print(f"  {msg}")
    print(f"{'='*60}")


def main():
    print("=" * 60)
    print("  adb-link PostgreSQL CRUD Integration Test")
    print("=" * 60)

    # ------------------------------------------------------------------
    # Step 1: Create database "test"
    # ------------------------------------------------------------------
    step("Step 1: Create database 'test'")

    # Drop if exists (ignore errors)
    # First terminate connections to the test database
    execute_sql(
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'test' AND pid <> pg_backend_pid();",
        database="postgres",
    )
    drop_result = execute_sql("DROP DATABASE IF EXISTS test;", database="postgres")
    print(f"  Drop existing: success={drop_result.get('success')}")

    result = execute_sql("CREATE DATABASE test;", database="postgres")
    assert_success(result, "CREATE DATABASE test")

    # ------------------------------------------------------------------
    # Step 2: Create tables
    # ------------------------------------------------------------------
    step("Step 2: Create tables")

    create_departments = """
    CREATE TABLE departments (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL UNIQUE,
        location VARCHAR(200),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    """
    result = execute_sql(create_departments, database=TEST_DB)
    assert_success(result, "CREATE TABLE departments")

    create_employees = """
    CREATE TABLE employees (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        email VARCHAR(200) UNIQUE,
        department_id INTEGER REFERENCES departments(id),
        salary NUMERIC(10, 2),
        hire_date DATE DEFAULT CURRENT_DATE,
        is_active BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    """
    result = execute_sql(create_employees, database=TEST_DB)
    assert_success(result, "CREATE TABLE employees")

    # ------------------------------------------------------------------
    # Step 3: INSERT data
    # ------------------------------------------------------------------
    step("Step 3: INSERT data")

    # Insert departments
    insert_depts = """
    INSERT INTO departments (name, location) VALUES
        ('Engineering', 'Building A, Floor 3'),
        ('Marketing', 'Building B, Floor 1'),
        ('Human Resources', 'Building A, Floor 1'),
        ('Finance', 'Building C, Floor 2');
    """
    result = execute_sql(insert_depts, database=TEST_DB)
    assert_success(result, "INSERT departments")

    # Insert employees
    insert_emps = """
    INSERT INTO employees (name, email, department_id, salary, hire_date, is_active) VALUES
        ('Alice Zhang', 'alice@example.com', 1, 95000.00, '2022-03-15', TRUE),
        ('Bob Wang', 'bob@example.com', 1, 88000.00, '2022-06-01', TRUE),
        ('Charlie Li', 'charlie@example.com', 2, 72000.00, '2023-01-10', TRUE),
        ('Diana Chen', 'diana@example.com', 3, 68000.00, '2023-04-20', TRUE),
        ('Edward Liu', 'edward@example.com', 4, 82000.00, '2021-11-01', TRUE),
        ('Fiona Xu', 'fiona@example.com', 1, 105000.00, '2020-08-15', TRUE),
        ('George Sun', 'george@example.com', 2, 76000.00, '2023-07-01', FALSE);
    """
    result = execute_sql(insert_emps, database=TEST_DB)
    assert_success(result, "INSERT employees")

    # ------------------------------------------------------------------
    # Step 4: SELECT data
    # ------------------------------------------------------------------
    step("Step 4: SELECT data")

    # Simple select
    result = execute_sql("SELECT * FROM departments ORDER BY id;", database=TEST_DB)
    assert_success(result, "SELECT departments")
    rows = result["data"]["rows"]
    print(f"  Departments count: {len(rows)}")
    assert len(rows) == 4, f"Expected 4 departments, got {len(rows)}"

    # Select with JOIN
    join_sql = """
    SELECT e.name, e.email, e.salary, d.name AS department
    FROM employees e
    JOIN departments d ON e.department_id = d.id
    WHERE e.is_active = TRUE
    ORDER BY e.salary DESC;
    """
    result = execute_sql(join_sql, database=TEST_DB)
    assert_success(result, "SELECT employees JOIN departments")
    rows = result["data"]["rows"]
    print(f"  Active employees: {len(rows)}")
    assert len(rows) == 6, f"Expected 6 active employees, got {len(rows)}"
    # Highest salary should be Fiona
    assert rows[0][0] == "Fiona Xu", f"Expected Fiona Xu as top earner, got {rows[0][0]}"
    print(f"  Top earner: {rows[0][0]} (${rows[0][2]})")

    # Aggregate query
    agg_sql = """
    SELECT d.name, COUNT(*) as emp_count, AVG(e.salary)::NUMERIC(10,2) as avg_salary
    FROM employees e
    JOIN departments d ON e.department_id = d.id
    WHERE e.is_active = TRUE
    GROUP BY d.name
    ORDER BY avg_salary DESC;
    """
    result = execute_sql(agg_sql, database=TEST_DB)
    assert_success(result, "SELECT aggregate by department")
    rows = result["data"]["rows"]
    print(f"  Department stats:")
    for row in rows:
        print(f"    {row[0]}: {row[1]} employees, avg salary ${row[2]}")

    # ------------------------------------------------------------------
    # Step 5: UPDATE data
    # ------------------------------------------------------------------
    step("Step 5: UPDATE data")

    # Give Engineering a 10% raise
    update_sql = """
    UPDATE employees
    SET salary = salary * 1.10
    WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering')
      AND is_active = TRUE;
    """
    result = execute_sql(update_sql, database=TEST_DB)
    assert_success(result, "UPDATE salary +10% for Engineering")

    # Verify the update
    verify_sql = """
    SELECT name, salary FROM employees
    WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering')
      AND is_active = TRUE
    ORDER BY name;
    """
    result = execute_sql(verify_sql, database=TEST_DB)
    assert_success(result, "Verify UPDATE")
    rows = result["data"]["rows"]
    print(f"  Engineering salaries after raise:")
    for row in rows:
        print(f"    {row[0]}: ${row[1]}")
    # Alice was 95000, now should be 104500
    alice_salary = float(rows[0][1])
    assert abs(alice_salary - 104500.0) < 0.01, f"Expected Alice salary ~104500, got {alice_salary}"

    # Update department location
    result = execute_sql(
        "UPDATE departments SET location = 'Building D, Floor 5' WHERE name = 'Engineering';",
        database=TEST_DB,
    )
    assert_success(result, "UPDATE department location")

    # ------------------------------------------------------------------
    # Step 6: DELETE data
    # ------------------------------------------------------------------
    step("Step 6: DELETE data")

    # Delete inactive employees
    result = execute_sql(
        "DELETE FROM employees WHERE is_active = FALSE;",
        database=TEST_DB,
    )
    assert_success(result, "DELETE inactive employees")

    # Verify deletion
    result = execute_sql("SELECT COUNT(*) FROM employees;", database=TEST_DB)
    assert_success(result, "Verify DELETE")
    count = result["data"]["rows"][0][0]
    print(f"  Remaining employees: {count}")
    assert int(count) == 6, f"Expected 6 remaining employees, got {count}"

    # Delete a specific employee
    result = execute_sql(
        "DELETE FROM employees WHERE email = 'edward@example.com';",
        database=TEST_DB,
    )
    assert_success(result, "DELETE specific employee")

    result = execute_sql("SELECT COUNT(*) FROM employees;", database=TEST_DB)
    assert_success(result, "Verify final count")
    count = result["data"]["rows"][0][0]
    print(f"  Final employee count: {count}")
    assert int(count) == 5, f"Expected 5 employees, got {count}"

    # ------------------------------------------------------------------
    # Step 7: Cleanup
    # ------------------------------------------------------------------
    step("Step 7: Cleanup (drop tables and database)")

    result = execute_sql("DROP TABLE IF EXISTS employees;", database=TEST_DB)
    assert_success(result, "DROP TABLE employees")

    result = execute_sql("DROP TABLE IF EXISTS departments;", database=TEST_DB)
    assert_success(result, "DROP TABLE departments")

    # Switch back to postgres db to drop test
    execute_sql(
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'test' AND pid <> pg_backend_pid();",
        database="postgres",
    )
    result = execute_sql("DROP DATABASE test;", database="postgres")
    assert_success(result, "DROP DATABASE test")

    # ------------------------------------------------------------------
    # Done
    # ------------------------------------------------------------------
    print("\n" + "=" * 60)
    print("  ALL TESTS PASSED!")
    print("=" * 60)


if __name__ == "__main__":
    main()
