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
        timeout(time: 45, unit: 'MINUTES')
    }
    parameters {
        string( name: 'DATA_NODE_BRANCH', defaultValue: '',
                description: '''Git branch, tag or hash of the vegaprotocol/data-node repository.
                    e.g. "develop", "v0.44.0" or commit hash. Default empty: use latests published version.''')
        string( name: 'VEGAWALLET_BRANCH', defaultValue: '',
                description: '''Git branch, tag or hash of the vegaprotocol/vegawallet repository.
                    e.g. "develop", "v0.9.0" or commit hash. Default empty: use latest published version.''')
        string( name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master',
                description: 'Git branch, tag or hash of the vegaprotocol/devops-infra repository')
        string( name: 'VEGATOOLS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/vegatools repository')
        string( name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/system-tests repository')
        string( name: 'PROTOS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/protos repository')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        DOCKER_IMAGE_TAG_LOCAL = "v-${ env.JOB_BASE_NAME.replaceAll('[^A-Za-z0-9\\._]','-') }-${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
        DOCKER_IMAGE_VEGA_CORE_LOCAL = "ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_LOCAL}"
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
                }
                echo "params (after injection)=${params}"
            }
        }

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
                    sh 'go mod download -x'
                }
            }
        }

        stage('Compile') {
            failFast true
            parallel {
                stage('Linux build') {
                    environment {
                        GOOS         = 'linux'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-linux-amd64'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Compile', script: '''
                                go build -v -o "${OUTPUT}" ./cmd/vega
                            '''
                            sh label: 'Sanity check', script: '''
                                file ${OUTPUT}
                                ${OUTPUT} version
                            '''
                        }
                    }
                }
                stage('MacOS build') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-darwin-amd64'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Compile', script: '''
                                go build -v -o "${OUTPUT}" ./cmd/vega
                            '''
                            sh label: 'Sanity check', script: '''
                                file ${OUTPUT}
                            '''
                        }
                    }
                }
                stage('Windows build') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/vega/vega-windows-amd64'
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Compile', script: '''
                                go build -v -o "${OUTPUT}" ./cmd/vega
                            '''
                            sh label: 'Sanity check', script: '''
                                file ${OUTPUT}
                            '''
                        }
                    }
                }
            }
        }

        stage('Build docker image') {
            environment {
                LINUX_BINARY = './cmd/vega/vega-linux-amd64'
            }
            options { retry(3) }
            steps {
                dir('vega') {
                    sh label: 'Copy binary', script: '''#!/bin/bash -e
                        mkdir -p docker/bin
                        cp -a "${LINUX_BINARY}" "docker/bin/vega"
                    '''
                    // Note: This docker image is used by publish stage
                    withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://ghcr.io"]) {
                        sh label: 'Build docker image', script: '''
                            docker build -t "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" docker/
                        '''
                    }
                    sh label: 'Cleanup', script: '''#!/bin/bash -e
                        rm -rf docker/bin
                    '''
                    sh label: 'Sanity check', script: '''
                        docker run --rm --entrypoint "" "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" vega version
                    '''
                }
            }
        }

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
                            sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
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
                                vegaCore: commitHash
                        }
                    }
                }
            }
        }

        stage('Tests') {
            parallel {
                stage('unit tests') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
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
                            sh 'go test -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                            junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                        }
                    }
                }
                stage('vega/integration tests') {
                    options { retry(3) }
                    steps {
                        dir('vega/integration') {
                            sh 'godog build -o integration.test && ./integration.test --format=junit:vega-integration-report.xml'
                            junit checksName: 'Integration Tests', testResults: 'vega-integration-report.xml'
                        }
                    }
                }
                stage('System Tests Network Smoke') {
                    steps {
                        script {
                            systemTestsCapsule ignoreFailure: !isPRBuild(),
                                vegaCore: commitHash,
                                dataNode: params.DATA_NODE_BRANCH,
                                vegawallet: params.VEGAWALLET_BRANCH,
                                vegatools: params.VEGATOOLS_BRANCH,
                                systemTests: params.SYSTEM_TESTS_BRANCH,
                                protos: params.PROTOS_BRANCH,
                                testMark: "network_infra_smoke"
                        }
                    }
                }
                stage('Capsule System Tests') {
                        steps {
                            script {
                                systemTestsCapsule vegaCore: commitHash,
                                    dataNode: params.DATA_NODE_BRANCH,
                                    vegawallet: params.VEGAWALLET_BRANCH,
                                    devopsInfra: params.DEVOPS_INFRA_BRANCH,
                                    vegatools: params.VEGATOOLS_BRANCH,
                                    systemTests: params.SYSTEM_TESTS_BRANCH,
                                    protos: params.PROTOS_BRANCH,
                                    ignoreFailure: !isPRBuild()

                            }
                        }
                }
            }
        }


        stage('Publish') {
            parallel {

                stage('docker image') {
                    when {
                        anyOf {
                            buildingTag()
                            branch 'develop'
                            // changeRequest() // uncomment only for testing
                        }
                    }
                    environment {
                        DOCKER_IMAGE_TAG_VERSIONED = "${ env.TAG_NAME ? env.TAG_NAME : env.BRANCH_NAME }"
                        DOCKER_IMAGE_VEGA_CORE_VERSIONED = "ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_VERSIONED}"
                        DOCKER_IMAGE_TAG_ALIAS = "${ env.TAG_NAME ? 'latest' : 'edge' }"
                        DOCKER_IMAGE_VEGA_CORE_ALIAS = "ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_ALIAS}"
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Tag new images', script: '''#!/bin/bash -e
                                docker image tag "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" "${DOCKER_IMAGE_VEGA_CORE_VERSIONED}"
                                docker image tag "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" "${DOCKER_IMAGE_VEGA_CORE_ALIAS}"
                            '''

                            withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://ghcr.io"]) {
                                sh label: 'Push docker images', script: '''
                                    docker push "${DOCKER_IMAGE_VEGA_CORE_VERSIONED}"
                                    docker push "${DOCKER_IMAGE_VEGA_CORE_ALIAS}"
                                '''
                            }
                            slackSend(
                                channel: "#tradingcore-notify",
                                color: "good",
                                message: ":docker: Vega Core » Published new docker image `${DOCKER_IMAGE_VEGA_CORE_VERSIONED}` aka `${DOCKER_IMAGE_VEGA_CORE_ALIAS}`",
                            )
                        }
                    }
                }

                stage('release to GitHub') {
                    when {
                        buildingTag()
                    }
                    environment {
                        RELEASE_URL = "https://github.com/vegaprotocol/vega/releases/tag/${TAG_NAME}"
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            script {
                                withGHCLI('credentialsId': 'github-vega-ci-bot-artifacts') {
                                    sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                                        [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''

                                        gh release view $TAG_NAME && gh release upload $TAG_NAME ./cmd/vega/vega-* \
                                            || gh release create $TAG_NAME $prerelease ./cmd/vega/vega-*
                                    '''
                                }
                            }
                            slackSend(
                                channel: "#tradingcore-notify",
                                color: "good",
                                message: ":rocket: Vega Core » Published new version to GitHub <${RELEASE_URL}|${TAG_NAME}>",
                            )
                        }
                    }
                }

                stage('Deploy to Devnet') {
                    when {
                        branch 'develop'
                    }
                    steps {
                        devnetDeploy vegaCore: commitHash,
                            wait: false
                    }
                }
            }
        }

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
        cleanup {
            retry(3) {
                sh label: 'Clean docker images', script: '''#!/bin/bash -e
                    [ -z "$(docker images -q "${DOCKER_IMAGE_VEGA_CORE_LOCAL}")" ] || docker rmi "${DOCKER_IMAGE_VEGA_CORE_LOCAL}"
                '''
            }
        }
    }
}
