@AWSPCAClusterIssuer
Feature: Issue certificates using an AWSPCAClusterIssuer 
  As a user of the aws-privateca-issuer
  I need to be able to issue certificates using an AWSPCAClusterIssuer

  Scenario Outline: Issue a certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | RSA    | RSA            |
      | RSA    | ECDSA          |
      | ECDSA  | RSA            |
      | ECDSA  | ECDSA          |

  Scenario Outline: Issue a CA certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    And I update my certificate spec to issue a CA certificate
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | RSA    | RSA            |
      | RSA    | ECDSA          |
      | ECDSA  | RSA            |
      | ECDSA  | ECDSA          |

  Scenario Outline: Issue a short lived certificate with a ClusterIssuer
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with duration of 20 hours
    And I update my certificate spec to issue a certificate with renew before of 5 hours
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | caType | certType       |
      | RSA    | RSA            |
      | RSA    | ECDSA          |
      | ECDSA  | RSA            |
      | ECDSA  | ECDSA          |

  @KubernetesSecrets
  Scenario Outline: Issue a certificate with a ClusterIssuer using a secret for AWS credentials
    Given I create a Secret with keys <accessKeyId> and <secretKeyId> for my AWS credentials
    And I create an AWSPCAClusterIssuer using a <caType> CA
    And I update my certificate spec to issue a <certType> certificate
    When I issue the certificate
    Then the certificate should be issued successfully

    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | RSA    | ECDSA          |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | RSA            |
      | AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | ECDSA  | ECDSA          |

    @KeySelectors
    Examples:
      | accessKeyId       | secretKeyId           | caType | certType       |
      | myKeyId           | mySecret              | RSA    | RSA            |
      | myKeyId           | mySecret              | RSA    | ECDSA          |
      | myKeyId           | mySecret              | ECDSA  | RSA            |
      | myKeyId           | mySecret              | ECDSA  | ECDSA          |

  @KeyUsage
  Scenario Outline: Issue a certificate with specific key usage
    Given I create an AWSPCAClusterIssuer using a <caType> CA
    And I update my certificate spec to issue a <certType> certificate
    And I update my certificate spec to issue a certificate with usages of <usage>
    When I issue the certificate
    Then the certificate should be issued successfully
    Then the certificate should be issued with usage <usage>

    Examples:
      | caType | certType | usage                    |
      | RSA    | RSA      | client_auth              |
      | RSA    | RSA      | server_auth              |
      | RSA    | RSA      | code_signing             |
      | RSA    | RSA      | ocsp_signing             |
      | RSA    | RSA      | any                      |
      | RSA    | RSA      | server_auth,client_auth  |
      | RSA    | RSA      | email protection         |
      | RSA    | RSA      | ipsec user               |
      | RSA    | RSA      | ipsec tunnel             |

  @CertificateRecovery
  Scenario: Issue a certificate with a non-existent issuer, is successfully issued after the issuer is created
    Given I create an AWSPCAClusterIssuer using a RSA CA
    And I delete the AWSPCAClusterIssuer
    And I update my certificate spec to issue a RSA certificate
    And I issue the certificate
    And the certificate request has been created
    And the certificate request has reason Pending and status False
    When I create an AWSPCAClusterIssuer using a RSA CA
    Then the certificate should be issued successfully

