@RoleAssumption
Feature: Issue certificates using role assumption
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates using role assumption

  Scenario: Issue a certificate with a ClusterIssuer using role assumption
    Given I create an AWSPCAClusterIssuer with role assumption
    And I update my certificate spec to issue a RSA certificate
    When I issue the certificate
    Then the certificate should be issued successfully

  Scenario: Issue a certificate with a namespaced Issuer using role assumption
    Given I create a namespace
    And I create an AWSPCAIssuer with role assumption
    And I update my certificate spec to issue a RSA certificate
    When I issue the certificate
    Then the certificate should be issued successfully
