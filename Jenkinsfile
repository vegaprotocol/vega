@Library('vega-shared-library') _

/* properties of scmVars (example):
    - GIT_BRANCH:PR-40-head
    - GIT_COMMIT:05a1c6fbe7d1ff87cfc40a011a63db574edad7e6
    - GIT_PREVIOUS_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_PREVIOUS_SUCCESSFUL_COMMIT:5d02b46fdb653f789e799ff6ad304baccc32cbf9
    - GIT_URL:https://github.com/vegaprotocol/data-node.git
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
        string( name: 'VEGA_CORE_BRANCH', defaultValue: '',
                description: '''Git branch, tag or hash of the vegaprotocol/vega repository.
                    e.g. "develop", "v0.44.0 or commit hash. Default empty: use latests published version.''')
        string( name: 'VEGAWALLET_BRANCH', defaultValue: '',
                description: '''Git branch, tag or hash of the vegaprotocol/vegawallet repository.
                    e.g. "develop", "v0.9.0" or commit hash. Default empty: use latest published version.''')
        string( name: 'ETHEREUM_EVENT_FORWARDER_BRANCH', defaultValue: '',
                description: '''Git branch, tag or hash of the vegaprotocol/ethereum-event-forwarder repository.
                    e.g. "main", "v0.44.0" or commit hash. Default empty: use latest published version.''')
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
        GO111MODULE = 'on'
        CGO_ENABLED  = '0'
        DOCKER_IMAGE_TAG_LOCAL = "j-${ env.JOB_BASE_NAME.replaceAll('[^A-Za-z0-9\\._]','-') }-${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
        DOCKER_IMAGE_NAME_LOCAL = "ghcr.io/vegaprotocol/data-node/data-node:${DOCKER_IMAGE_TAG_LOCAL}"
    }

    stages {
        stage('Config') {
            steps {
                cleanWs()
                sh 'printenv'
                echo "${params}"
		script {
                    params = pr.injectPRParams()
                }
                echo "params (after injection)=${params}"
            }
        }
        stage('Git Clone') {
            options { retry(3) }
            steps {
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

        stage('Dependencies') {
            options { retry(3) }
            steps {
                sh 'go mod download -x'
            }
        }

        stage('Build') {
            environment {
                LDFLAGS      = "-X main.CLIVersion=${version} -X main.CLIVersionHash=${versionHash}"
            }
            failFast true
            parallel {
                stage('Linux') {
                    environment {
                        GOOS         = 'linux'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-linux-amd64'
                    }
                    options { retry(3) }
                    steps {
                        sh label: 'Compile', script: '''
                            go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
                        '''
                        sh label: 'Sanity check', script: '''
                            file ${OUTPUT}
                            ${OUTPUT} version
                        '''
                    }
                }
                stage('MacOS') {
                    environment {
                        GOOS         = 'darwin'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-darwin-amd64'
                    }
                    options { retry(3) }
                    steps {
                        sh label: 'Compile', script: '''
                            go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
                        '''
                        sh label: 'Sanity check', script: '''
                            file ${OUTPUT}
                        '''
                    }
                }
                stage('Windows') {
                    environment {
                        GOOS         = 'windows'
                        GOARCH       = 'amd64'
                        OUTPUT       = './cmd/data-node/data-node-windows-amd64'
                    }
                    options { retry(3) }
                    steps {
                        sh label: 'Compile', script: '''
                            go build -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/data-node
                        '''
                        sh label: 'Sanity check', script: '''
                            file ${OUTPUT}
                        '''
                    }
                }
            }
        }

        // this task needs to run after builds
        stage('Build docker image') {
            environment {
                LINUX_BINARY = './cmd/data-node/data-node-linux-amd64'
            }
            options { retry(3) }
            steps {
                sh label: 'Copy binary', script: '''#!/bin/bash -e
                    mkdir -p docker/bin
                    cp -a "${LINUX_BINARY}" "docker/bin/data-node"
                '''
                withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://ghcr.io"]) {
                    sh label: 'Build docker image', script: '''
                        docker build -t "${DOCKER_IMAGE_NAME_LOCAL}" docker/
                    '''
                }
                sh label: 'Cleanup', script: '''#!/bin/bash -e
                    rm -rf docker/bin
                '''
                sh label: 'Sanity check', script: '''
                    docker run --rm "${DOCKER_IMAGE_NAME_LOCAL}" version
                '''
            }
        }

        stage('Run linters') {
            parallel {
                stage('check print') {
                    options { retry(3) }
                    steps {
                        sh 'make print_check'
                    }
                }
                stage('shellcheck') {
                    options { retry(3) }
                    steps {
                        sh "git ls-files '*.sh'"
                        sh "git ls-files '*.sh' | xargs shellcheck"
                    }
                }
                stage('70+ linters') {
                    steps {
                        sh '''#!/bin/bash -e
                            golangci-lint run -v --config .golangci.toml
                        '''
                    }
                }
                stage('yamllint') {
                    options { retry(3) }
                    steps {
                        sh "git ls-files '*.yml' '*.yaml'"
                        sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
                    }
                }
                stage('python files') {
                    options { retry(3) }
                    steps {
                        sh "git ls-files '*.py'"
                        sh "git ls-files '*.py' | xargs flake8"
                        sh "git ls-files '*.py' | xargs black -l 79 --check --diff"
                    }
                }
                stage('json format') {
                    options { retry(3) }
                    steps {
                        sh "git ls-files '*.json'"
                        sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
                    }
                }
                stage('markdown spellcheck') {
                    options { retry(3) }
                    steps {
                        sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
                    }
                }
            }
        }

        stage('Run tests') {
            parallel {
                stage('unit tests') {
                    options { retry(3) }
                    steps {
                        sh 'go test -v $(go list ./...) 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
                        junit checksName: 'Unit Tests', testResults: 'vega-unit-test-report.xml'
                    }
                }
                stage('unit tests with race') {
                    environment {
                        CGO_ENABLED = '1'
                    }
                    options { retry(3) }
                    steps {
                        sh 'go test -v -race $(go list ./...) 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                        junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                    }
                }
                stage('System Tests') {
                    steps {
                        script {
                            systemTests ignoreFailure: !isPRBuild(),
                                vegaCore: params.VEGA_CORE_BRANCH,
                                dataNode: commitHash,
                                vegawallet: params.VEGAWALLET_BRANCH,
                                ethereumEventForwarder: params.ETHEREUM_EVENT_FORWARDER_BRANCH,
                                devopsInfra: params.DEVOPS_INFRA_BRANCH,
                                vegatools: params.VEGATOOLS_BRANCH,
                                systemTests: params.SYSTEM_TESTS_BRANCH,
                                protos: params.PROTOS_BRANCH
                        }
                    }
                }
                stage('LNL System Tests') {
                    steps {
                        script {
                            systemTestsLNL ignoreFailure: !isPRBuild(),
                                vegaCore: params.VEGA_CORE_BRANCH,
                                dataNode: commitHash,
                                vegawallet: params.VEGAWALLET_BRANCH,
                                ethereumEventForwarder: params.ETHEREUM_EVENT_FORWARDER_BRANCH,
                                devopsInfra: params.DEVOPS_INFRA_BRANCH,
                                vegatools: params.VEGATOOLS_BRANCH,
                                systemTests: params.SYSTEM_TESTS_BRANCH,
                                protos: params.PROTOS_BRANCH
                        }
                    }
                }
		stage('Capsule System Tests') {
                        steps {
                            script {
                                systemTestsCapsule vegaCore: params.VEGA_CORE_BRANCH,
                                    dataNode: commitHash,
                                    vegawallet: params.VEGAWALLET_BRANCH,
                                    devopsInfra: params.DEVOPS_INFRA_BRANCH,
                                    vegatools: params.VEGATOOLS_BRANCH,
                                    systemTests: params.SYSTEM_TESTS_BRANCH,
                                    protos: params.PROTOS_BRANCH,
                                    ignoreFailure: true // Will be changed when stable

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
                        DOCKER_IMAGE_NAME_VERSIONED = "ghcr.io/vegaprotocol/data-node/data-node:${DOCKER_IMAGE_TAG_VERSIONED}"
                        DOCKER_IMAGE_TAG_ALIAS = "${ env.TAG_NAME ? 'latest' : 'edge' }"
                        DOCKER_IMAGE_NAME_ALIAS = "ghcr.io/vegaprotocol/data-node/data-node:${DOCKER_IMAGE_TAG_ALIAS}"
                    }
                    options { retry(3) }
                    steps {
                        sh label: 'Tag new images', script: '''#!/bin/bash -e
                            docker image tag "${DOCKER_IMAGE_NAME_LOCAL}" "${DOCKER_IMAGE_NAME_VERSIONED}"
                            docker image tag "${DOCKER_IMAGE_NAME_LOCAL}" "${DOCKER_IMAGE_NAME_ALIAS}"
                        '''

                        withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://ghcr.io"]) {
                            sh label: 'Push docker images', script: '''
                                docker push "${DOCKER_IMAGE_NAME_VERSIONED}"
                                docker push "${DOCKER_IMAGE_NAME_ALIAS}"
                            '''
                        }
                        slackSend(
                            channel: "#tradingcore-notify",
                            color: "good",
                            message: ":docker: Data-Node » Published new docker image `${DOCKER_IMAGE_NAME_VERSIONED}` aka `${DOCKER_IMAGE_NAME_ALIAS}`",
                        )
                    }
                }

                stage('release to GitHub') {
                    when {
                        buildingTag()
                    }
                    environment {
                        RELEASE_URL = "https://github.com/vegaprotocol/data-node/releases/tag/${TAG_NAME}"
                    }
                    options { retry(3) }
                    steps {
                        withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', passwordVariable: 'TOKEN', usernameVariable:'USER')]) {
                            // Workaround for user input:
                            //  - global configuration: 'gh config set prompt disabled'
                            sh label: 'Log in to a Gihub with CI', script: '''
                                echo ${TOKEN} | gh auth login --with-token -h github.com
                            '''
                        }
                        sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                            [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''
                            gh release create $TAG_NAME $prerelease ./cmd/data-node/data-node-*
                        '''
                        slackSend(
                            channel: "#tradingcore-notify",
                            color: "good",
                            message: ":rocket: Data-Node » Published new version to GitHub <${RELEASE_URL}|${TAG_NAME}>",
                        )
                    }
                    post {
                        always  {
                            retry(3) {
                                script {
                                    sh label: 'Log out from Github', script: '''
                                        gh auth logout -h github.com
                                    '''
                                }
                            }
                        }
                    }
                }

                stage('[TODO] deploy to Devnet') {
                    when {
                        branch 'develop'
                    }
                    steps {
                        echo 'Deploying to Devnet....'
                        echo 'Run basic tests on Devnet network ...'
                    }
                }
            }
        }

    }
    post {
        success {
            retry(3) {
                script {
                    slack.slackSendCISuccess name: 'Data-Node CI', channel: '#tradingcore-notify'
                }
            }
        }
        unsuccessful {
            retry(3) {
                script {
                    slack.slackSendCIFailure name: 'Data-Node CI', channel: '#tradingcore-notify'
                }
            }
        }
        cleanup {
            retry(3) {
                sh label: 'Clean docker images', script: '''
                    docker rmi "${DOCKER_IMAGE_NAME_LOCAL}"
                '''
            }
        }
    }
}
