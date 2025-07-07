# Multi-Currency Support

## Overview

This feature adds support for multiple currencies in accounts and transactions.

## Design

- Add a currency field to the Account and Transaction domain models.
- Update services to handle currency conversions where necessary.
- Ensure all monetary operations respect the currency type.
- Add validation to prevent mixing currencies in operations.

## Implementation Plan

1. Modify domain models to include currency.
2. Update repositories and database schema.
3. Update service methods to handle currency.
4. Add currency conversion service if needed.
5. Add tests for multi-currency scenarios.

## Notes

- Default currency will be USD unless specified.
- Currency codes will follow ISO 4217 standard.
