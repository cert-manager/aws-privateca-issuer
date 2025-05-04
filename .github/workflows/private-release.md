name: Release
on:
  push:
    tags:
      - "*"
  workflow_dispatch:

env:
  PRIVATE_REGISTRY: 105154636954.dkr.ecr.us-east-1.amazonaws.com
  EKS_USER_AGENT: aws-privateca-connector-for-kubernetes-eks-addon

jobs:
  build:
    name: release
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
      - name: Get release version
        id: tag
        uses: divyansh-gupta/action-get-tag@727a6f0a561be04e09013531e73a3983a65e3479
      - name: Setup Dockerx
        uses: docker/setup-buildx-action@v1
      - name: Setup AWS Credentials
        uses: aws-actions/configure-aws-credentials@master
        with:
          role-to-assume: arn:aws:iam::105154636954:role/GithubActionsPublishRole-prod-us-east-1
          aws-region: us-east-1
      - name: Login to Staging ECR
        uses: docker/login-action@v1
        with:
          registry: ${{ env.PRIVATE_REGISTRY }}
        env:
          AWS_REGION: us-east-1
      - name: Setup Push to Staging ECR
        run: |
          export TAG_BASE=${{ env.PRIVATE_REGISTRY }}/eks/$(echo $GITHUB_REPOSITORY | sed s#.*/##)
          echo TAG_BASE=$TAG_BASE >> $GITHUB_ENV
      - name: Build and push arm image
        uses: docker/build-push-action@v6
        with:
          build-args: |
            pkg_version=${{ steps.tag.outputs.tag }}
            user_agent=${{ env.EKS_USER_AGENT }}
          context: .
          platforms: linux/arm64
          provenance: false
          tags: |
            ${{ env.TAG_BASE }}:${{steps.tag.outputs.tag}}-arm64
          push: true
      - name: Build and push amd image
        uses: docker/build-push-action@v6
        with:
          build-args: |
            pkg_version=${{ steps.tag.outputs.tag }}
            user_agent=${{ env.EKS_USER_AGENT }}
          context: .
          platforms: linux/amd64
          provenance: false
          tags: |
            ${{ env.TAG_BASE }}:${{steps.tag.outputs.tag}}-amd64
          push: true
      - name: Build and push manifest list
        uses: docker/build-push-action@v6
        with:
          build-args: |
            pkg_version=${{ steps.tag.outputs.tag }}
            user_agent=${{ env.EKS_USER_AGENT }}
          context: .
          platforms: linux/amd64,linux/arm64
          provenance: false
          tags: |
            ${{ env.TAG_BASE }}:latest
            ${{ env.TAG_BASE }}:${{steps.tag.outputs.tag}}
          push: true
      - name: Publish Helm chart
        uses: divyansh-gupta/helm-gh-pages@12f5926e622ccae035cf5a3bb8d67ae6db7dc4b7
        with:
          token: ${{ secrets.CR_PAT }}
          linting: "off"
          app_version: ${{ steps.tag.outputs.tag }}
