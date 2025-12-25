---
icon: material/test-tube
---

# 🧪 Testing Stripe Connect

Before going live, test your Connect integration for account creation, identity verification, and payouts.

Use testing to make sure your *Connect* (Connect is Stripe's solution for multi-party businesses, such as marketplace or software platforms, to route payments between sellers, customers, and other recipients) integration handles different flows correctly. You can use [Sandboxes](https://docs.stripe.com/sandboxes.md) to simulate live mode while taking advantage of Stripe-provided special tokens to use in your tests. See the [payments testing guide](https://docs.stripe.com/testing.md) for more information on testing charges, disputes, and so on.

> #### Testing capabilities
>
> Sandboxes and test mode might not enforce some capabilities. In certain cases, they can allow an account to perform capability-dependent actions even when the associated capability’s `status` isn’t `active`.

## 📌 Create test accounts

You can create multiple test accounts with different [account types](https://docs.stripe.com/connect/accounts.md) or [controller properties](https://docs.stripe.com/connect/migrate-to-controller-properties.md) that you want to test.

You can create test accounts using the [Accounts API](https://docs.stripe.com/api/accounts/create.md) or in the [Stripe Dashboard](https://docs.stripe.com/connect/dashboard/managing-individual-accounts.md#creating-accounts).

Use `000-000` as the SMS code when prompted for test accounts.

## 📌 Test the OAuth flow

You can test your OAuth integration with connected accounts that use a Stripe-hosted Dashboard using your test `client_id`.

Your test `client_id` is `ca_FkyHCg7X8mlvCUdMDao4mMxagUfhIwXb`. You can find this in your [Connect OAuth settings](https://dashboard.stripe.com/settings/connect/onboarding-options/oauth).

Your test `client_id` allows you to:

- Set your `redirect_uri` to a non-HTTPS URL
- Set your `redirect_uri` to **localhost**
- Force-skip the account form instead of having to fill in an entire account application (Stripe Dashboard accounts only)
- Get test access tokens for connected accounts

To test the [OAuth](https://docs.stripe.com/connect/oauth-standard-accounts.md) flow, create a new account after clicking the OAuth link. You can also test connecting an existing Stripe account only if the email address is different from your platform account.

## 📌 Identity verification

Verification is a crucial component for onboarding accounts. Use our dedicated [guide to testing verification](https://docs.stripe.com/connect/testing-verification.md).

After creating a test connected account, you can use tokens to test different verification statuses to make sure you’re handling different requirements and account states. You can use the following tokens to test verification with test accounts.

### Test dates of birth

Use these dates of birth (DOB) to trigger certain verification conditions.

| DOB          | Type                                                                                                                                                                                                                                                      |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `1901-01-01` | Successful date of birth match. Any other DOB results in a no-match.                                                                                                                                                                                      |
| `1902-01-01` | Successful, immediate date of birth match. The verification result is returned directly in the response, not as part of a *webhook* (A webhook is a real-time push notification sent to your application as a JSON payload through HTTPS requests) event. |
| `1900-01-01` | This DOB triggers an Office of Foreign Assets Control (OFAC) alert.                                                                                                                                                                                       |

### Test addresses

Use these addresses for `line1` to trigger certain verification conditions. You must pass in legitimate values for the `city`, `state`, and `postal_code` arguments.

| Token                      | Type                                                                          |
| -------------------------- | ----------------------------------------------------------------------------- |
| `address_full_match`​      | Successful address match.                                                     |
| `address_full_match_sync`​ | Successful, immediate address match.                                          |
| `address_no_match`         | Unsuccessful address match likely to trigger requirements in `currently_due`. |
| `address_line1_no_match`   | Unsuccessful address match likely to trigger requirements in `currently_due`  |

### Test personal ID numbers

Use these personal ID numbers for the [individual.id_number](https://docs.stripe.com/api/accounts/create.md#create_account-individual-id_number) attribute on the `Account` or the [id_number](https://docs.stripe.com/api/persons/create.md#create_person-id_number) attribute on the `Person` object to trigger certain verification conditions.

| Number      | Type                                                                                                                                                                                                                                                  |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `000000000` | Successful ID number match. **0000** also works for SSN last 4 verification.                                                                                                                                                                          |
| `111111111` | Unsuccessful ID number match (identity mismatch) likely to trigger requirements in `currently_due`.                                                                                                                                                   |
| `222222222` | Successful, immediate ID number match. The verification result is returned directly in the response, not as part of a *webhook* (A webhook is a real-time push notification sent to your application as a JSON payload through HTTPS requests) event. |

### Test identity documents

For testing, use test images or file tokens instead of uploading your own test IDs. For details, refer to [Uploading a file](https://docs.stripe.com/connect/handling-api-verification.md#upload-a-file).

### Test document images

You can use a [verified image](https://d37ugbyn3rpeym.cloudfront.net/docs/identity/success.png) that causes the ID number to match successfully. You can use an [unverified image](https://d37ugbyn3rpeym.cloudfront.net/docs/identity/failed.png) that causes a mismatch on the ID number, leading to `currently_due` requirements.

> Test images take precedence over test ID numbers. If you upload a verified image, the ID number matching succeeds, even if you also provide an unsuccessful test ID value. Similarly, an unverified image automatically fails ID matching regardless of the value of other test artefacts.

### Test file tokens

Use these file tokens to trigger certain identity verification conditions.

| Token                            | Type                                                                            |
| -------------------------------- | ------------------------------------------------------------------------------- |
| `file_identity_document_success` | Uses the verified image and marks that document requirement as satisfied.       |
| `file_identity_document_failure` | Uses the unverified image and marks that document requirement as not satisfied. |

## Business information verification

### Business address validation

In some countries, the business address associated with your connected account must be validated before charges, *payouts* (A payout is the transfer of funds to an external account, usually a bank account, in the form of a deposit), or both can be enabled on the connected account.

### Test business addresses

Use these addresses for `line1` to trigger certain validation conditions. You must pass in legitimate values for the `city`, `state`, and `postal_code` arguments.

Make sure you start with an address token that has the least permissive validation condition you want to test for. This is because you can’t use an address token that has a more restrictive validation condition than the previous token used. For example, if you provided `address_full_match` to have both charges and payouts enabled, you can’t disable payouts or charges afterward by changing the token to an invalid one. You can work around this by creating a new account with the relevant token.

| Token                    | Type                                                                                                                                                                                                                                |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `address_full_match`​    | Both charges and payouts are enabled on the account.                                                                                                                                                                                |
| `address_no_match`​      | Only charges are enabled on the account. Since validation failed on the `line1` attribute, it becomes listed again in the [requirements](https://docs.stripe.com/api/accounts/object.md#account_object-requirements) hash.          |
| `address_line1_no_match` | Neither charges nor payouts are enabled on the account. Since validation failed, the address attributes become listed again in the [requirements](https://docs.stripe.com/api/accounts/object.md#account_object-requirements) hash. |

### Test business tax IDs

Use these business tax ID numbers for [company.tax_id](https://docs.stripe.com/api/accounts/create.md#create_account-company-tax_id) to trigger certain verification conditions. The test behaviour might change depending on the Connected Account countries and the regulations in those countries. Depending on the country’s regulation, a valid tax document can mark tax ID verified in these countries.

| Number      | Type                                                                                                                                                                                                                                                           |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `000000000` | Successful business ID number match.                                                                                                                                                                                                                           |
| `000000001` | Successful business ID number match as a non-profit.                                                                                                                                                                                                           |
| `111111111` | Unsuccessful business ID number match (identity mismatch).                                                                                                                                                                                                     |
| `111111112` | Unsuccessful business ID number match (tax ID not issued).                                                                                                                                                                                                     |
| `222222222` | Successful, immediate business ID number match. The verification result is returned directly in the response, not as part of a *webhook* (A webhook is a real-time push notification sent to your application as a JSON payload through HTTPS requests) event. |

### Test directorship verification

Stripe performs directorship verification by comparing the list of directors on the `Account` object against a list retrieved from local registries. If the country requires it, you can trigger verification for an `Account` object by using these tokens for the [first_name](https://docs.stripe.com/api/persons/object.md#person_object-first_name) attribute on the associated `Person` and setting the [relationship.director](https://docs.stripe.com/api/persons/object.md#person_object-relationship-director) attribute on the `Person` to true.

| Token                 | Type                                                                                                                                    |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `mismatch_director`   | Unsuccessful match due to a mismatched name. This can trigger a `verification_directors_mismatch` verification error.                   |
| `missing_director`    | Unsuccessful match due to directors missing on the account. This can trigger a `verification_missing_directors` verification error.     |
| `extraneous_director` | Unsuccessful match due to too many directors on the account. This can trigger a `verification_extraneous_directors` verification error. |

The verification errors can trigger if multiple directors on the `Account` object use these magic tokens.

### Test company name verification

Trigger company name verification for an `Account` object by using this token for the [company.name](https://docs.stripe.com/api/accounts/object.md#account_object-company-name) attribute.

| Token                      | Type                                                             |
| -------------------------- | ---------------------------------------------------------------- |
| `mismatch_business_name`   | Unsuccessful match due to a mismatched business name.            |
| `disallowed_name`          | Unsuccessful match due to a generic or well-known business name. |
| `match_name_relationships` | Successful match of the business name.                           |
| `match_name_only`          | Unsuccessful match due to a business name discrepancy.           |

### Test statement descriptor verification

Trigger statement descriptor verification for an `Account` object by using this token for the [settings.payments.statement_descriptor](https://docs.stripe.com/api/accounts/object.md#account_object-settings-payments-statement_descriptor) attribute.

| Token        | Type                                                                            |
| ------------ | ------------------------------------------------------------------------------- |
| `mismatch`   | Trigger an `invalid_statement_descriptor_business_mismatch` verification error. |
| `disallowed` | Trigger an `invalid_statement_descriptor_denylisted` verification error.        |

Trigger statement descriptor prefix verification for an `Account` object by using this token for the [settings.card_payments.statement_descriptor_prefix](https://docs.stripe.com/api/accounts/object.md#account_object-settings-card_payments-statement_descriptor_prefix) attribute.

| Token        | Type                                                                            |
| ------------ | ------------------------------------------------------------------------------- |
| `mismatch`   | Trigger an `invalid_statement_descriptor_prefix_mismatch` verification error.   |
| `disallowed` | Trigger an `invalid_statement_descriptor_prefix_denylisted` verification error. |

### Test business URL verification

Trigger URL verification for an `Account` object by using this token for the [business_profile.url](https://docs.stripe.com/api/accounts/object.md#account_object-business_profile-url) attribute.

| Token                                  | Type                                                                                 |
| -------------------------------------- | ------------------------------------------------------------------------------------ |
| `https://disallowed.stripe.com`        | Trigger an `invalid_url_denylisted` verification error.                              |
| `https://geoblocked.stripe.com`        | Trigger an `invalid_url_website_inaccessible_geoblocked` verification error.         |
| `https://problem.stripe.com`           | Trigger an `invalid_url_website_other` verification error.                           |
| `https://missing.stripe.com`           | Trigger an `invalid_url_website_incomplete` verification error.                      |
| `https://mismatch.stripe.com`          | Trigger an `invalid_url_website_business_information_mismatch` verification error.   |
| `https://passwordprotected.stripe.com` | Trigger an `invalid_url_website_inaccessible_password_protected` verification error. |
| `https://accessible.stripe.com`        | Trigger a successful validation of the URL.                                          |
| `https://underconstruction.stripe.com` | Trigger an `invalid_url_website_incomplete_under_construction` verification error.   |
| `https://inaccessible.stripe.com`      | Trigger an `invalid_url_website_inaccessible` verification error.                    |

### Test capability disabled reasons

Trigger assignment of a specific [requirements.disabled_reason](https://docs.stripe.com/api/capabilities/object.md#capability_object-requirements-disabled_reason) to all of an `Account` object’s inactive `Capability` objects by using this token for the account’s [business_profile.url](https://docs.stripe.com/api/accounts/object.md#account_object-business_profile-url) attribute.

| Token                           | Type                                                                                                                                                                                                   |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `https://inactivity.stripe.com` | Set an account as inactive and pause all verifications for it. Set the disabled reason for any inactive capabilities to `paused.inactivity` (`rejected.other` for API versions prior to `2024-06-20`). |

### Test Doing Business As (DBA) verification

Trigger DBA verification for an `Account` object by using this token for the [business_profile.name](https://docs.stripe.com/api/accounts/object.md#account_object-business_profile-name) attribute.

| Token            | Type                                                                      |
| ---------------- | ------------------------------------------------------------------------- |
| `disallowed_dba` | Trigger an `invalid_business_profile_name_denylisted` verification error. |
| `invalid_dba`    | Trigger an `invalid_business_profile_name` verification error.            |

### Test product description verification

Trigger product description verification for an `Account` object by using this token for the [business_profile.product_description](https://docs.stripe.com/api/accounts/object.md#account_object-business_profile-product_description) attribute.

| Token         | Type                                                               |
| ------------- | ------------------------------------------------------------------ |
| `require_url` | Trigger an `invalid_url_web_presence_detected` verification error. |

### Test phone number validation

Clear phone number validation for an `Account` object by using this token for the following attributes:

- [business_profile.support_phone](https://docs.stripe.com/api/accounts/object.md#account_object-business_profile-support_phone)
- [company.phone](https://docs.stripe.com/api/accounts/object.md#account_object-company-phone)
- [individual.phone](https://docs.stripe.com/api/accounts/object.md#account_object-individual-phone)

Clear phone number validation for a `Person` object by using this token for the [phone](https://docs.stripe.com/api/persons/object.md#person_object-phone) attribute.

| Token        | Type                  |
| ------------ | --------------------- |
| `0000000000` | Successful validation |

## Trigger or advance verification

### Trigger cards

Use these card numbers to trigger various conditions when you’re testing both requirements and tiered verification. For the trigger actions to work, you must use these cards with a Connect charge by setting [on_behalf_of](https://docs.stripe.com/connect/separate-charges-and-transfers.md#settlement-merchant), or creating the charge [directly on the connected account](https://docs.stripe.com/connect/direct-charges.md).

| Number           | Token                              | Type                                                                  |
| ---------------- | ---------------------------------- | --------------------------------------------------------------------- |
| 4000000000004202 | `tok_visa_triggerNextRequirements` | Changes the next set of eventually due requirements to currently due. |
| 4000000000004210 | `tok_visa_triggerChargeBlock`      | Triggers a charge block.                                              |
| 4000000000004236 | `tok_visa_triggerPayoutBlock`      | Triggers a payout block.                                              |

#### Trigger next requirements

Live mode can require additional verification information when a connected account processes a certain amount of volume. This card sets any additional verification information to be required immediately. If no additional information is required, nothing appears.

#### Trigger a charge or payout block

If required information isn’t provided by the deadline, Stripe disables the connected account’s charges or payouts. These cards disable the connected account and move any currently due requirements to overdue. These cards have no effect until an account provides the initial information that’s required to enable charges and payouts.

### Trigger bank account ownership verification

Connected accounts in the United States and India are subject to [Bank account ownership verification](https://support.stripe.com/questions/bank-account-ownership-verification). You can complete this verification by uploading supporting documents with the Connect Dashboard or with the API through the [documents[bank_account_ownership_verification]](https://docs.stripe.com/api/accounts/update.md#update_account-documents-bank_account_ownership_verification-files) hash.

While you’re testing, you can simulate the US bank account ownership verification process. Use the following test bank account numbers to trigger the verification process. One number presumes successful verification and the other prompts you to upload test images or file tokens to complete the verification process. These test accounts are only available for US accounts.

| Routing     | Account        | Type                                                                                                          |
| ----------- | -------------- | ------------------------------------------------------------------------------------------------------------- |
| `110000000` | `000999999991` | Triggers and completes the bank account ownership verification process after a short delay                    |
| `110000000` | `000999999992` | Triggers the bank account ownership verification process after a short delay and requests for document upload |

## Add funds to Stripe balance

To test [adding funds](https://docs.stripe.com/connect/top-ups.md) to your Stripe balance from a bank account in the Dashboard, [create a sandbox](https://docs.stripe.com/sandboxes/dashboard/manage.md#create-a-sandbox) and select the desired test bank account in the dropdown menu within the **Add to balance** window. You can simulate success or failure due to insufficient funds.

To test adding funds in the API, use the following test bank tokens as the source while you’re testing. Each token simulates a specific kind of event.

| Token                                 | Type                                            |
| ------------------------------------- | ----------------------------------------------- |
| `btok_us_verified`                    | Successful                                      |
| `btok_us_verified_noAccount`          | Unsuccessful with a `no_account` code           |
| `btok_us_verified_accountClosed`      | Unsuccessful with an `account_closed` code      |
| `btok_us_verified_insufficientFunds`  | Unsuccessful with an `insufficient_funds` code  |
| `btok_us_verified_debitNotAuthorized` | Unsuccessful with a `debit_not_authorized` code |
| `btok_us_verified_invalidCurrency`    | Unsuccessful with an `invalid_currency` code    |

## Payouts

Use the following test bank and debit card numbers to trigger certain events during [payout](https://docs.stripe.com/connect/payouts-connected-accounts.md) testing. You can only use these values while testing with test secret keys.

Test payouts simulate a live payout but aren’t processed with the bank. Test accounts with Stripe Dashboard access always have payouts enabled, as long as valid external bank information and other relevant conditions are met, and never requires real identity verification.

> You can’t use test bank and debit card numbers in the Stripe Dashboard on a live mode connected account. If you’ve entered your bank account information on a live mode account, you can still use a sandbox, and test payouts will simulate a live payout without processing actual money.

### Bank numbers

Use these test bank account numbers to test payouts. You can only use them with test secret keys.

### Debit card numbers

Use these test debit card numbers to test payouts to a debit card. These can only be used with test secret keys.

#### United States

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000056655665556 | `tok_visa_debit_us_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000056655665572 | `tok_visa_debit_us_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000056755665555 | `tok_visa_debit_us_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5200828282828210 | `tok_mastercard_debit_us_transferSuccess`    | Mastercard debit. Payout succeeds.                        |
| 6011981111111113 | `tok_discover_debit_us_transferSuccess`      | Discover debit. Payout succeeds.                          |

#### Canada

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000051240000005 | `tok_visa_debit_ca_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000051240000021 | `tok_visa_debit_ca_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000051240000039 | `tok_visa_debit_ca_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5510121240000006 | `tok_mastercard_debit_ca_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Singapore

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000057020000008 | `tok_visa_debit_sg_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000057020000016 | `tok_visa_debit_sg_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000057020000024 | `tok_visa_debit_sg_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 2227200000000009 | `tok_mastercard_debit_sg_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Australia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000050360000019 | `tok_visa_debit_au_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000050360000027 | `tok_visa_debit_au_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000050360000035 | `tok_visa_debit_au_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 4000000360000006 | `tok_visa_credit_au`                         | Visa credit. Card Not Supported (invalid card type).      |
| 5555050360000023 | `tok_mastercard_debit_au_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### United Kingdom

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000058260000203 | `tok_visa_debit_gb_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000058260000211 | `tok_visa_debit_gb_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000058260000229 | `tok_visa_debit_gb_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555058260000100 | `tok_mastercard_debit_gb_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Austria

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000050400000003 | `tok_visa_debit_at_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000050400000011 | `tok_visa_debit_at_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000050400000029 | `tok_visa_debit_at_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555050400000009 | `tok_mastercard_debit_at_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Belgium

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000050560000009 | `tok_visa_debit_be_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000050560000017 | `tok_visa_debit_be_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000050560000025 | `tok_visa_debit_be_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555050560000005 | `tok_mastercard_debit_be_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Croatia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000051910000004 | `tok_visa_debit_hr_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000051910000012 | `tok_visa_debit_hr_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000051910000020 | `tok_visa_debit_hr_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555051910000000 | `tok_mastercard_debit_hr_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Cyprus

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000051960000003 | `tok_visa_debit_cy_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000051960000011 | `tok_visa_debit_cy_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000051960000029 | `tok_visa_debit_cy_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555051960000009 | `tok_mastercard_debit_cy_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Estonia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052330000004 | `tok_visa_debit_ee_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052330000012 | `tok_visa_debit_ee_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052330000020 | `tok_visa_debit_ee_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052330000000 | `tok_mastercard_debit_ee_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Finland

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052460000006 | `tok_visa_debit_fi_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052460000014 | `tok_visa_debit_fi_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052460000022 | `tok_visa_debit_fi_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052460000002 | `tok_mastercard_debit_fi_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### France

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052500000008 | `tok_visa_debit_fr_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052500000016 | `tok_visa_debit_fr_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052500000024 | `tok_visa_debit_fr_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052500000004 | `tok_mastercard_debit_fr_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Germany

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052760000037 | `tok_visa_debit_de_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052760000011 | `tok_visa_debit_de_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052760000029 | `tok_visa_debit_de_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052760000009 | `tok_mastercard_debit_de_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Greece

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000053000000001 | `tok_visa_debit_gr_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000053000000019 | `tok_visa_debit_gr_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000053000000027 | `tok_visa_debit_gr_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555053000000007 | `tok_mastercard_debit_gr_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Ireland

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000053720000000 | `tok_visa_debit_ie_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000053720000018 | `tok_visa_debit_ie_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000053720000026 | `tok_visa_debit_ie_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555053720000006 | `tok_mastercard_debit_ie_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Italy

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000053800000037 | `tok_visa_debit_it_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000053800000011 | `tok_visa_debit_it_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000053800000029 | `tok_visa_debit_it_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555053800000009 | `tok_mastercard_debit_it_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Latvia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000054280000000 | `tok_visa_debit_lv_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000054280000018 | `tok_visa_debit_lv_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000054280000026 | `tok_visa_debit_lv_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555054280000006 | `tok_mastercard_debit_lv_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Lithuania

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000054400000005 | `tok_visa_debit_lt_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000054400000013 | `tok_visa_debit_lt_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000054400000021 | `tok_visa_debit_lt_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555054400000001 | `tok_mastercard_debit_lt_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Luxembourg

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000054420000001 | `tok_visa_debit_lu_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000054420000019 | `tok_visa_debit_lu_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000054420000027 | `tok_visa_debit_lu_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555054420000007 | `tok_mastercard_debit_lu_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Malta

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000054700000002 | `tok_visa_debit_mt_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000054700000010 | `tok_visa_debit_mt_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000054700000028 | `tok_visa_debit_mt_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555054700000008 | `tok_mastercard_debit_mt_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Netherlands

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000055280000007 | `tok_visa_debit_nl_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000055280000015 | `tok_visa_debit_nl_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000055280000023 | `tok_visa_debit_nl_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555055280000003 | `tok_mastercard_debit_nl_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Portugal

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000056200000002 | `tok_visa_debit_pt_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000056200000010 | `tok_visa_debit_pt_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000056200000028 | `tok_visa_debit_pt_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555056200000008 | `tok_mastercard_debit_pt_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Slovakia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000057030000006 | `tok_visa_debit_sk_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000057030000014 | `tok_visa_debit_sk_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000057030000022 | `tok_visa_debit_sk_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555057030000002 | `tok_mastercard_debit_sk_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Slovenia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000057050000001 | `tok_visa_debit_si_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000057050000019 | `tok_visa_debit_si_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000057050000027 | `tok_visa_debit_si_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555057050000007 | `tok_mastercard_debit_si_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Spain

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000057240000036 | `tok_visa_debit_es_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000057240000010 | `tok_visa_debit_es_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000057240000028 | `tok_visa_debit_es_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555057240000008 | `tok_mastercard_debit_es_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Denmark

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052080000006 | `tok_visa_debit_dk_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052080000014 | `tok_visa_debit_dk_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052080000022 | `tok_visa_debit_dk_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052080000002 | `tok_mastercard_debit_dk_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Malaysia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000054580000031 | `tok_visa_debit_my_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000054580000015 | `tok_visa_debit_my_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000054580000023 | `tok_visa_debit_my_instantPayoutUnsupported` | Visa debit. Card is not eligible for Instant Payouts.     |
| 5555054580000003 | `tok_mastercard_debit_my_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### New Zealand

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000055540000003 | `tok_visa_debit_nz_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000055540000011 | `tok_visa_debit_nz_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000055540000029 | `tok_visa_debit_nz_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555055540000165 | `tok_mastercard_debit_nz_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Norway

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000055780000002 | `tok_visa_debit_no_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000055780000010 | `tok_visa_debit_no_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000055780000028 | `tok_visa_debit_no_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555055780000008 | `tok_mastercard_debit_no_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Sweden

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000057520000003 | `tok_visa_debit_se_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000057520000011 | `tok_visa_debit_se_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000057520000029 | `tok_visa_debit_se_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555057520000009 | `tok_mastercard_debit_se_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Czechia

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000052030000007 | `tok_visa_debit_cz_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000052030000015 | `tok_visa_debit_cz_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000052030000023 | `tok_visa_debit_cz_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555052030000003 | `tok_mastercard_debit_cz_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Hungary

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000053480000000 | `tok_visa_debit_hu_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000053480000018 | `tok_visa_debit_hu_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000053480000026 | `tok_visa_debit_hu_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555053480000006 | `tok_mastercard_debit_hu_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Poland

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000056160000000 | `tok_visa_debit_pl_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000056160000018 | `tok_visa_debit_pl_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000056160000026 | `tok_visa_debit_pl_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555056160000006 | `tok_mastercard_debit_pl_transferSuccess`    | Mastercard debit. Payout succeeds.                        |

#### Romania

| Number           | Token                                        | Type                                                      |
| ---------------- | -------------------------------------------- | --------------------------------------------------------- |
| 4000056420000030 | `tok_visa_debit_ro_transferSuccess`          | Visa debit. Payout succeeds.                              |
| 4000056420000014 | `tok_visa_debit_ro_transferFail`             | Visa debit. Payout fails with a `could_not_process` code. |
| 4000056420000022 | `tok_visa_debit_ro_instantPayoutUnsupported` | Visa debit. Card isn’t eligible for Instant Payouts.      |
| 5555056420000002 | `tok_mastercard_debit_ro_transferSuccess`    | Mastercard debit. Payout succeeds.                        |
