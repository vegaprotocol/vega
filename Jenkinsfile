/* groovylint-disable DuplicateStringLiteral, LineLength, NestedBlockDepth */
@Library('vega-shared-library') _

/* properties of scmVars (example):
    - GIT_BRANCH:PR-40-head
    - GIT_COMMIT:05a1c6fbe7d1ff87cfc40a011a63db574edad7e6
    - GIT_PREVIOUS_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_PREVIOUS_SUCCESSFUL_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_URL:https://github.com/vegaprotocol/vega.git
*/
def scmVars = null
def version = 'UNKNOWN'
def versionHash = 'UNKNOWN'
def commitHash = 'UNKNOWN'


pipeline {
    agent any
    options {
        skipDefaultCheckout true
        timestamps()
        timeout(time: isPRBuild() ? 50 : 120, unit: 'MINUTES')
    }
    parameters {
        string( name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/system-tests repository')
        string( name: 'VEGACAPSULE_BRANCH', defaultValue: '',
                description: 'Git branch, tag or hash of the vegaprotocol/vegacapsule repository')
        string( name: 'VEGATOOLS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/vegatools repository')
        string( name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master',
                description: 'Git branch, tag or hash of the vegaprotocol/devops-infra repository')
        string( name: 'DEVOPSSCRIPTS_BRANCH', defaultValue: 'main',
                description: 'Git branch, tag or hash of the vegaprotocol/devopsscripts repository')
        string( name: 'VEGA_MARKET_SIM_BRANCH', defaultValue: '',
                description: 'Git branch, tag or hash of the vegaprotocol/vega-market-sim repository')
        string( name: 'JENKINS_SHARED_LIB_BRANCH', defaultValue: 'main',
                description: 'Git branch, tag or hash of the vegaprotocol/jenkins-shared-library repository')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        BUILD_UID="${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
        DOCKER_CONFIG="${env.WORKSPACE}/docker-home"
        DOCKER_BUILD_ARCH = "${ isPRBuild() ? 'linux/amd64' : 'linux/arm64,linux/amd64' }"
        DOCKER_IMAGE_TAG = "${ env.TAG_NAME ? 'latest' : env.BRANCH_NAME }"
        DOCKER_VEGA_BUILDER_NAME="vega-${BUILD_UID}"
        DOCKER_DATANODE_BUILDER_NAME="data-node-${BUILD_UID}"
        DOCKER_VEGAWALLET_BUILDER_NAME="vegawallet-${BUILD_UID}"
        DOCKER_VEGA_BUILDER_CACHE="${env.WORKSPACE}/docker-cache-vega"
        DOCKER_DATANODE_BUILDER_CACHE="${env.WORKSPACE}/docker-cache-data-node"
        DOCKER_VEGAWALLET_BUILDER_CACHE="${env.WORKSPACE}/docker-cache-vegawallet"
    }

    stages {
        stage('Config') {
            steps {
                cleanWs()
                sh 'printenv'
                echo "params=${params}"
                echo "isPRBuild=${isPRBuild()}"
                script {
                    params = pr.injectPRParams()
                    if (env.CHANGE_URL) {
                        prParams = pr.getData(url: env.CHANGE_URL, prFields:['headRepositoryOwner', 'headRepository'])
                        echo "prParams = ${prParams}"
                        originRepo = "${prParams.headRepositoryOwner.login}/${prParams.headRepository.name}"
                    } else {
                        originRepo = "vegaprotocol/vega"
                    }
                }
                echo "params (after injection)=${params}"
            }
        }

        //
        // Begin PREPARE
        //
        stage('Git clone') {
            options { retry(3) }
            steps {
                dir('vega') {
                    script {
                        scmVars = checkout(scm)
                        versionHash = sh (returnStdout: true, script: "echo \"${scmVars.GIT_COMMIT}\"|cut -b1-8").trim()
                        version = sh (returnStdout: true, script: "git describe --tags 2>/dev/null || echo ${versionHash}").trim()
                        commitHash = getCommitHash()
                    }
                    echo "scmVars=${scmVars}"
                    echo "commitHash=${commitHash}"
                }
            }
        }

        stage('Dependencies') {
            options { retry(3) }
            steps {
                dir('vega') {
                    sh '''#!/bin/bash -e
                        go mod download -x
                    '''
                }
            }
        }

        stage('Docker login') {
            options { retry(3) }
            steps {
                withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD')]) {
                    sh label: 'docker login ghcr.io', script: '''#!/bin/bash -e
                        echo "${PASSWORD}" | docker login --username ${USERNAME} --password-stdin ghcr.io
                    '''
                }
            }
        }
        //
        // End PREPARE
        //

        //
        // Begin COMPILE
        //
        stage('Compile') {
            options { retry(3) }
            steps {
                sh 'printenv'
                dir('vega') {
                    sh label: 'Compile', script: """#!/bin/bash -e
                        go build -v \
                            -o ../build/ \
                            ./cmd/vega \
                            ./cmd/data-node \
                            ./cmd/vegawallet
                    """
                    sh label: 'check for modifications', script: 'git diff'
                }
                dir("build") {
                    sh label: 'list files', script: '''#!/bin/bash -e
                        pwd
                        ls -lah
                    '''
                    sh label: 'Sanity check', script: '''#!/bin/bash -e
                        file *
                    '''
                    sh label: 'get version', script: '''#!/bin/bash -e
                        ./vega version
                        ./data-node version
                        ./vegawallet version
                    '''
                }
            }
        }
        //
        // End COMPILE
        //

        //
        // Begin LINTERS
        //
        stage('Linters') {
            parallel {
                stage('linters') {
                    steps {
                        dir('vega') {
                            sh '''#!/bin/bash -e
                                golangci-lint run -v --config .golangci.toml
                            '''
                        }
                    }
                }
                stage('shellcheck') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.sh'"
                            sh "git ls-files '*.sh' | xargs shellcheck"
                        }
                    }
                }
                stage('yamllint') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.yml' '*.yaml'"
                            sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 200}}}'"
                        }
                    }
                }
                stage('json format') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh "git ls-files '*.json'"
                            sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
                        }
                    }
                }
                stage('markdown spellcheck') {
                    environment {
                        FORCE_COLOR = '1'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            ansiColor('xterm') {
                                sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
                            }
                        }
                        sh 'printenv'
                    }
                }
                stage('approbation') {
                    when {
                        anyOf {
                            branch 'develop'
                            branch 'main'
                            branch 'master'
                        }
                    }
                    steps {
                        script {
                            runApprobation ignoreFailure: !isPRBuild(),
                                originRepo: originRepo,
                                vegaVersion: commitHash
                        }
                    }
                }
                stage('protos') {
                    environment {
                        GOPATH = "${env.WORKSPACE}/GOPATH"
                        GOBIN = "${env.GOPATH}/bin"
                        PATH = "${env.GOBIN}:${env.PATH}"
                    }
                    stages {
                        stage('Install dependencies') {
                            // We are using specific tools versions
                            // Please use exactly the same versions when modifying protos
                            options { retry(3) }
                            steps {
                                dir('vega') {
                                    sh 'printenv'
                                    sh './script/gettools.sh'
                                }
                            }
                        }
                        stage('buf lint') {
                            options { retry(3) }
                            steps {
                                dir('vega') {
                                    sh '''#!/bin/bash -e
                                        buf lint
                                    '''
                                }
                            }
                            post {
                                failure {
                                    sh 'printenv'
                                    echo "params=${params}"
                                    sh 'buf --version'
                                    sh 'which buf'
                                    sh 'git diff'
                                }
                            }
                        }
                        stage('proto check') {
                            options { retry(3) }
                            steps {
                                sh label: 'copy vega repo', script: '''#!/bin/bash -e
                                        cp -r ./vega ./vega-proto-check
                                    '''
                                dir('vega-proto-check') {
                                    sh '''#!/bin/bash -e
                                        make proto_check
                                    '''
                                }
                                sh label: 'remove vega copy', script: '''#!/bin/bash -e
                                        rm -rf ./vega-proto-check
                                    '''
                            }
                            post {
                                failure {
                                    sh 'printenv'
                                    echo "params=${params}"
                                    sh 'buf --version'
                                    sh 'which buf'
                                    sh 'git diff'
                                }
                            }
                        }
                    }
                }
                stage('create docker builders') {
                    steps {
                        sh label: 'vega builder', script: """#!/bin/bash -e
                            docker buildx create --bootstrap --name ${DOCKER_VEGA_BUILDER_NAME}
                        """
                        sh label: 'data-node builder', script: """#!/bin/bash -e
                            docker buildx create --bootstrap --name ${DOCKER_DATANODE_BUILDER_NAME}
                        """
                        sh label: 'vegawallet builder', script: """#!/bin/bash -e
                            docker buildx create --bootstrap --name ${DOCKER_VEGAWALLET_BUILDER_NAME}
                        """
                        sh 'docker buildx ls'
                        sh label: 'create cache directory for docker buildx', script: """#!/bin/bash -e
                            mkdir -p '${DOCKER_VEGA_BUILDER_CACHE}'
                            mkdir -p '${DOCKER_DATANODE_BUILDER_CACHE}'
                            mkdir -p '${DOCKER_VEGAWALLET_BUILDER_CACHE}'
                        """
                    }
                }  // docker builders
            }
        }
        //
        // End LINTERS
        //

        //
        // Begin TESTS
        //
        stage('Tests') {
            environment {
                DOCKER_IMAGE_TAG_VERSION = "${ env.TAG_NAME ?: versionHash }"
            }
            parallel {
                stage('unit tests') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test  -timeout 30m -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
                            junit checksName: 'Unit Tests', testResults: 'vega-unit-test-report.xml'
                        }
                    }
                }
                stage('unit tests with race') {
                    environment {
                        CGO_ENABLED = 1
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test -timeout 30m  -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                            junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                        }
                    }
                }
                stage('vega/integration tests') {
                    options { retry(3) }
                    steps {
                        dir('vega/core/integration') {
                            sh 'godog build -o integration.test && ./integration.test --format=junit:vega-integration-report.xml'
                            junit checksName: 'Integration Tests', testResults: 'vega-integration-report.xml'
                        }
                    }
                }
                stage('Vega Market Sim') {
                    when {
                        anyOf {
                            branch 'develop'
                            expression {
                                params.VEGA_MARKET_SIM_BRANCH
                            }
                        }
                    }
                    steps {
                        script {
                            vegaMarketSim ignoreFailure: true,
                                timeout: 45,
                                originRepo: originRepo,
                                vegaVersion: commitHash,
                                vegaMarketSim: params.VEGA_MARKET_SIM_BRANCH,
                                jenkinsSharedLib: params.JENKINS_SHARED_LIB_BRANCH
                        }
                    }
                }
                stage('System Tests') {
                    steps {
                        script {
                            systemTestsCapsule ignoreFailure: !isPRBuild(),
                                timeout: 30,
                                originRepo: originRepo,
                                vegaVersion: commitHash,
                                systemTests: params.SYSTEM_TESTS_BRANCH,
                                vegacapsule: params.VEGACAPSULE_BRANCH,
                                vegatools: params.VEGATOOLS_BRANCH,
                                devopsInfra: params.DEVOPS_INFRA_BRANCH,
                                devopsScripts: params.DEVOPSSCRIPTS_BRANCH,
                                jenkinsSharedLib: params.JENKINS_SHARED_LIB_BRANCH
                        }
                    }
                }
                stage('mocks check') {
                    steps {
                        sh label: 'copy vega repo', script: '''#!/bin/bash -e
                                cp -r ./vega ./vega-mocks-check
                            '''
                        dir('vega-mocks-check') {
                            sh '''#!/bin/bash -e
                                make mocks_check
                            '''
                        }
                        sh label: 'remove vega copy', script: '''#!/bin/bash -e
                                rm -rf ./vega-mocks-check
                            '''
                    }
                    post {
                        failure {
                            sh 'printenv'
                            echo "params=${params}"
                            dir('vega') {
                                sh 'git diff'
                            }
                        }
                    }
                }

                //
                // Build docker images during system-tests
                //
                stage('build vega docker image') {
                    steps {
                        dir('vega') {
                            sh 'printenv'
                            sh label: 'build vega docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_VEGA_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/vega.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_VERSION} \
                                    -t ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG} \
                                    --cache-to type=local,mode=max,dest='${DOCKER_VEGA_BUILDER_CACHE}' \
                                    .
                            """
                        }
                    }
                }
                stage('build data-node docker image') {
                    steps {
                        dir('vega') {
                            sh 'printenv'
                            sh label: 'build data-node docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_DATANODE_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/data-node.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/data-node:${DOCKER_IMAGE_TAG_VERSION} \
                                    -t ghcr.io/vegaprotocol/vega/data-node:${DOCKER_IMAGE_TAG} \
                                    --cache-to type=local,mode=max,dest='${DOCKER_DATANODE_BUILDER_CACHE}' \
                                    .
                            """
                        }
                    }
                }
                stage('build vegawallet docker image') {
                    steps {
                        dir('vega') {
                            sh 'printenv'
                            sh label: 'build vegawallet docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_VEGAWALLET_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/vegawallet.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/vegawallet:${DOCKER_IMAGE_TAG_VERSION} \
                                    -t ghcr.io/vegaprotocol/vega/vegawallet:${DOCKER_IMAGE_TAG} \
                                    --cache-to type=local,mode=max,dest='${DOCKER_VEGAWALLET_BUILDER_CACHE}' \
                                    .
                            """
                        }
                    }
                }
            }
        }
        //
        // End TESTS
        //

        //
        // Begin PUBLISH
        //
        stage('Publish') {
            environment {
                DOCKER_IMAGE_TAG_VERSION = "${ env.TAG_NAME ?: versionHash }"
            }
            parallel {
                stage('vega docker image') {
                    when {
                        branch 'develop'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'publish vega docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_VEGA_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/vega.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG} \
                                    -t ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_VERSION} \
                                    --cache-from type=local,src='${DOCKER_VEGA_BUILDER_CACHE}' \
                                    --push \
                                    .
                            """
                        }
                    }
                    post {
                        failure {
                            sh 'printenv'
                            echo "params=${params}"
                            sh 'docker buildx ls'
                        }
                    }
                }
                stage('data-node docker image') {
                    when {
                        branch 'develop'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'publish data-node docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_DATANODE_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/data-node.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/data-node:${DOCKER_IMAGE_TAG} \
                                    -t ghcr.io/vegaprotocol/vega/data-node:${DOCKER_IMAGE_TAG_VERSION} \
                                    --cache-from type=local,src='${DOCKER_DATANODE_BUILDER_CACHE}' \
                                    --push \
                                    .
                            """
                        }
                    }
                    post {
                        failure {
                            sh 'printenv'
                            echo "params=${params}"
                            sh 'docker buildx ls'
                        }
                    }
                }
                stage('vegawallet docker image') {
                    when {
                        branch 'develop'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'publish vegawallet docker image', script: """#!/bin/bash -e
                                docker buildx build \
                                    --builder ${DOCKER_VEGAWALLET_BUILDER_NAME} \
                                    --platform=${DOCKER_BUILD_ARCH} \
                                    -f docker/vegawallet.dockerfile \
                                    -t ghcr.io/vegaprotocol/vega/vegawallet:${DOCKER_IMAGE_TAG} \
                                    -t ghcr.io/vegaprotocol/vega/vegawallet:${DOCKER_IMAGE_TAG_VERSION} \
                                    --cache-from type=local,src='${DOCKER_VEGAWALLET_BUILDER_CACHE}' \
                                    --push \
                                    .
                            """
                        }
                    }
                    post {
                        failure {
                            sh 'printenv'
                            echo "params=${params}"
                            sh 'docker buildx ls'
                        }
                    }
                }

                stage('development binary for vegacapsule') {
                    when {
                        branch 'develop'
                    }
                    environment {
                        AWS_REGION = 'eu-west-2'
                    }
                    steps {
                        dir('build-linux-amd64') {
                            script {
                                vegaS3Ops = usernamePassword(
                                    credentialsId: 'vegacapsule-s3-operations',
                                    passwordVariable: 'AWS_ACCESS_KEY_ID',
                                    usernameVariable: 'AWS_SECRET_ACCESS_KEY'
                                )
                                bucketName = string(
                                    credentialsId: 'vegacapsule-s3-bucket-name',
                                    variable: 'VEGACAPSULE_S3_BUCKET_NAME'
                                )
                                withCredentials([vegaS3Ops, bucketName]) {
                                    try {
                                        sh label: 'Upload vega binary to S3', script: '''
                                            aws s3 cp ./vega s3://''' + env.VEGACAPSULE_S3_BUCKET_NAME + '''/bin/vega-linux-amd64-''' + versionHash + '''
                                        '''
                                    } catch(err) {
                                        print(err)
                                    }
                                    try {
                                        sh label: 'Upload data-node binary to S3', script: '''
                                            aws s3 cp ./data-node s3://''' + env.VEGACAPSULE_S3_BUCKET_NAME + '''/bin/data-node-linux-amd64-''' + versionHash + '''
                                        '''
                                    } catch(err) {
                                        print(err)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        //
        // End PUBLISH
        //

        //
        // Begin DEVNET deploy
        //
        stage('Deploy to Devnet') {
            when {
                branch 'develop'
            }
            steps {
                devnetDeploy vegaVersion: versionHash,
                    wait: false,
                    jenkinsSharedLib: params.JENKINS_SHARED_LIB_BRANCH
            }
        }
        //
        // End DEVNET deploy
        //

    }
    post {
        success {
            retry(3) {
                script {
                    slack.slackSendCISuccess name: 'Vega Core CI', channel: '#tradingcore-notify'
                }
            }
        }
        unsuccessful {
            retry(3) {
                script {
                    slack.slackSendCIFailure name: 'Vega Core CI', channel: '#tradingcore-notify'
                }
            }
        }
        always {
            retry(3) {
                sh label: 'destroy vega docker builder',
                returnStatus: true,  // ignore exit code
                script: """#!/bin/bash -e
                    docker buildx rm --force ${DOCKER_VEGA_BUILDER_NAME}
                """
                sh label: 'destroy data-node docker builder',
                returnStatus: true,  // ignore exit code
                script: """#!/bin/bash -e
                    docker buildx rm --force ${DOCKER_DATANODE_BUILDER_NAME}
                """
                sh label: 'destroy vegawallet docker builder',
                returnStatus: true,  // ignore exit code
                script: """#!/bin/bash -e
                    docker buildx rm --force ${DOCKER_VEGAWALLET_BUILDER_NAME}
                """
                sh label: 'docker logout ghcr.io',
                returnStatus: true,  // ignore exit code
                script: '''#!/bin/bash -e
                    docker logout ghcr.io
                '''
            }
            cleanWs()
        }
    }
}
