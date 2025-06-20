@AWSPCAIssuer
Feature: Issue certificates with specific key usages
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates with specific key usages

  Background: Create unique namespace and credentials
    Given I create a namespace
    And I create a Secret with keys AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY for my AWS credentials
@KeyUsage
  Scenario Outline: Issue certificate with specific usage
    Given I create an AWSPCAIssuer using a <caType> CA
    When I issue a <certType> certificate with usage <usage>
    Then the certificate should be issued successfully
    Then the certificate should be issued with usage <usage>

    Examples:
      | caType | certType | usage                    |
      | RSA    | RSA      | client_auth              |
      | RSA    | RSA      | server_auth              |
      | RSA    | RSA      | code_signing             |
      | RSA    | RSA      | ocsp_signing             |
      | RSA    | RSA      | any                      |
      | RSA    | RSA      | client_auth,server_auth  |
