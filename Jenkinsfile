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


pipeline {
    agent { label 'general' }
    options {
        skipDefaultCheckout true
        timestamps()
        timeout(time: 30, unit: 'MINUTES')
    }
    parameters {
        string(name: 'SYSTEM_TESTS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/system-tests repository')
        string(name: 'DATA_NODE_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/data-node repository')
        string(name: 'DEVOPS_INFRA_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/devops-infra repository')
        string(name: 'SPECS_INTERNAL_BRANCH', defaultValue: 'master', description: 'Git branch name of the vegaprotocol/specs-internal repository')
        string(name: 'PROTOS_BRANCH', defaultValue: 'develop', description: 'Git branch name of the vegaprotocol/protos repository')
        string(name: 'SYSTEM_TESTS_VALIDATOR_NODE_COUNT', defaultValue: '2', description: 'Number of validator nodes when running system-tests')
        string(name: 'SYSTEM_TESTS_NON_VALIDATOR_NODE_COUNT', defaultValue: '1', description: 'Number of non-validator nodes when running system-tests')
        string(name: 'SYSTEM_TESTS_TEST_FUNCTION', defaultValue: '', description: 'Run only a tests with a specified function name. This is actually a "pytest -k $TEST_FUNCTION_NAME" command-line argument, see more: https://docs.pytest.org/en/stable/usage.html')
        string(name: 'SYSTEM_TESTS_TEST_DIRECTORY', defaultValue: 'CoreTesting/bvt', description: 'Run tests from files in this directory and all sub-directories')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        SLACK_MESSAGE = "Vega Core CI » <${RUN_DISPLAY_URL}|Jenkins ${BRANCH_NAME} Job>${ env.CHANGE_URL ? " » <${CHANGE_URL}|GitHub PR #${CHANGE_ID}>" : '' }"
        // Note: make sure the tag name is not too long
        // Reason: it is used by system-tests for hostnames in dockerised vega, and
        //         there is a limit of 64 characters for hostname
        DOCKER_IMAGE_TAG_LOCAL = "v-${ env.JOB_BASE_NAME.replaceAll('[^A-Za-z0-9\\._]','-') }-${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
        DOCKER_IMAGE_VEGA_CORE_LOCAL = "docker.pkg.github.com/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_LOCAL}"
        DOCKER_IMAGE_DATA_NODE_LOCAL = "docker.pkg.github.com/vegaprotocol/data-node/data-node:${DOCKER_IMAGE_TAG_LOCAL}"
    }

    stages {
        stage('Git clone') {
            parallel {
                stage('vega core') {
                    options { retry(3) }
                    steps {
                        sh 'printenv'
                        echo "${params}"
                        dir('vega') {
                            script {
                                scmVars = checkout(scm)
                                versionHash = sh (returnStdout: true, script: "echo \"${scmVars.GIT_COMMIT}\"|cut -b1-8").trim()
                                version = sh (returnStdout: true, script: "git describe --tags 2>/dev/null || echo ${versionHash}").trim()
                            }
                        }
                    }
                }
                stage('data-node') {
                    options { retry(3) }
                    steps {
                        dir('data-node') {
                            git branch: "${params.DATA_NODE_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/data-node.git'
                        }
                    }
                }
                stage('specs-internal') {
                    options { retry(3) }
                    steps {
                        dir('specs-internal') {
                            git branch: "${params.SPECS_INTERNAL_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/specs-internal.git'
                        }
                    }
                }
                stage('protos') {
                    options { retry(3) }
                    steps {
                        dir('protos') {
                            git branch: "${params.PROTOS_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/protos.git'
                        }
                    }
                }
                stage('system-tests') {
                    options { retry(3) }
                    steps {
                        dir('system-tests') {
                            git branch: "${params.SYSTEM_TESTS_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/system-tests.git'
                        }
                    }
                }
                stage('devops-infra') {
                    options { retry(3) }
                    steps {
                        dir('devops-infra') {
                            git branch: "${params.DEVOPS_INFRA_BRANCH}", credentialsId: 'vega-ci-bot', url: 'git@github.com:vegaprotocol/devops-infra.git'
                        }
                    }
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
            environment {
                LDFLAGS      = "-X main.CLIVersion=${version} -X main.CLIVersionHash=${versionHash}"
            }
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
                                go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
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
                                go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
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
                                go build -v -o "${OUTPUT}" -ldflags "${LDFLAGS}" ./cmd/vega
                            '''
                            sh label: 'Sanity check', script: '''
                                file ${OUTPUT}
                            '''
                        }
                    }
                }
            }
        }

        stage(' ') {
            failFast true
            parallel {
                // this task needs to run after builds
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
                            // Note: This docker image is used by system-tests and publish stage
                            withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
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
            }
        }

        stage('Linters') {
            parallel {
                stage('static check') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'staticcheck -checks "all,-SA1019,-ST1000,-ST1021" ./...'
                        }
                    }
                }
                stage('go vet') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go vet ./...'
                        }
                    }
                }
                stage('check print') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'make print_check'
                        }
                    }
                }
                stage('misspell') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'golangci-lint run --allow-parallel-runners --disable-all --enable misspell'
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
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
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
                stage('specs-internal qa-scenarios') {
                    options { retry(3) }
                    steps {
                        dir('vega/integration') {
                            sh 'godog build -o qa_integration.test && ./qa_integration.test --format=junit:specs-internal-qa-scenarios-report.xml ../../specs-internal/qa-scenarios/'
                            junit checksName: 'Specs Tests (specs-internal)', testResults: 'specs-internal-qa-scenarios-report.xml'
                        }
                    }
                }
                stage('system-tests') {
                    environment {
                        SYSTEM_TESTS_PORTBASE = "${ Integer.parseInt(env.EXECUTOR_NUMBER) * 1000 + 1000}"
                        SYSTEM_TESTS_DOCKER_IMAGE_TAG = "${DOCKER_IMAGE_TAG_LOCAL}"
                        VEGA_CORE_IMAGE_TAG = "${DOCKER_IMAGE_TAG_LOCAL}"
                        DATA_NODE_IMAGE_TAG = "${ params.DATA_NODE_BRANCH == 'develop' ? 'develop' : env.DOCKER_IMAGE_TAG_LOCAL }"
                        VALIDATOR_NODE_COUNT = "${params.SYSTEM_TESTS_VALIDATOR_NODE_COUNT}"
                        NON_VALIDATOR_NODE_COUNT = "${params.SYSTEM_TESTS_NON_VALIDATOR_NODE_COUNT}"
                        TEST_FUNCTION = "${params.SYSTEM_TESTS_TEST_FUNCTION}"
                        TEST_DIRECTORY = "${params.SYSTEM_TESTS_TEST_DIRECTORY}"
                        DOCKER_GOCACHE = "${env.GOCACHE}"
                    }
                    stages {
                        stage('check') {
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'Check setup', script: '''
                                        make check
                                    '''
                                }
                            }
                        }
                        stage('docker pull') {
                            options { retry(3) }
                            steps {
                                dir('system-tests/scripts') {
                                    withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
                                        sh 'make prepare-docker-pull'
                                    }
                                }
                            }
                        }
                        stage('Build Data-Node') {
                            when { expression { env.DATA_NODE_IMAGE_TAG != 'develop'} }
                            options { retry(3) }
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'Build data-node app', script: '''
                                        make build-data-node
                                    '''
                                    sh label: 'Build data-node container', script: '''
                                        make build-data-node-docker-image
                                    '''
                                }
                            }
                        }
                        stage('Prepare tests') {
                            options { retry(3) }
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'build test container', script: '''
                                        make prepare-test-docker-image
                                    '''
                                    sh label: 'make proto', script: '''
                                        make build-test-proto
                                    '''
                                }
                            }
                        }
                        stage('Start dockerised-vega') {
                            options {
                                retry(2)
                                timeout(time: 10, unit: 'MINUTES')
                            }
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'make sure dockerised-vega is not running', script: '''
                                        make stop-dockerised-vega
                                    '''
                                    withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
                                        sh label: 'start dockerised-vega', script: '''
                                            make start-dockerised-vega
                                        '''
                                    }
                                }
                            }
                        }
                        stage('Run system-tests') {
                            steps {
                                dir('system-tests/scripts') {
                                    sh label: 'run system-tests', script: '''
                                        make run-tests || touch ../build/test-reports/system-test-results.xml
                                    '''
                                }
                                junit checksName: 'System Tests', testResults: 'system-tests/build/test-reports/system-test-results.xml'
                            }
                        }
                    }
                    post {
                        always  {
                            retry(3) {
                                script {
                                    dir('system-tests/scripts') {
                                        sh label: 'print logs from all the containers', script: '''
                                            make logs
                                        '''
                                        sh label: 'stop dockerised-vega', script: '''
                                            make stop-dockerised-vega
                                        '''
                                    }
                                }
                            }
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
                        DOCKER_IMAGE_VEGA_CORE_VERSIONED = "docker.pkg.github.com/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_VERSIONED}"
                        DOCKER_IMAGE_TAG_ALIAS = "${ env.TAG_NAME ? 'latest' : 'edge' }"
                        DOCKER_IMAGE_VEGA_CORE_ALIAS = "docker.pkg.github.com/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG_ALIAS}"
                    }
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh label: 'Tag new images', script: '''#!/bin/bash -e
                                docker image tag "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" "${DOCKER_IMAGE_VEGA_CORE_VERSIONED}"
                                docker image tag "${DOCKER_IMAGE_VEGA_CORE_LOCAL}" "${DOCKER_IMAGE_VEGA_CORE_ALIAS}"
                            '''

                            withDockerRegistry([credentialsId: 'github-vega-ci-bot-artifacts', url: "https://docker.pkg.github.com"]) {
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
                            withCredentials([usernamePassword(credentialsId: 'github-vega-ci-bot-artifacts', passwordVariable: 'TOKEN', usernameVariable:'USER')]) {
                                // Workaround for user input:
                                //  - global configuration: 'gh config set prompt disabled'
                                sh label: 'Log in to a Gihub with CI', script: '''
                                    echo ${TOKEN} | gh auth login --with-token -h github.com
                                '''
                            }
                            sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                                [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''
                                gh release create $TAG_NAME $prerelease ./cmd/vega/vega-*
                            '''
                            slackSend(
                                channel: "#tradingcore-notify",
                                color: "good",
                                message: ":rocket: Vega Core » Published new version to GitHub <${RELEASE_URL}|${TAG_NAME}>",
                            )
                        }
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
                    options { retry(3) }
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
                slackSend(channel: "#tradingcore-notify", color: "good", message: ":white_check_mark: ${SLACK_MESSAGE} (${currentBuild.durationString.minus(' and counting')})")
            }
        }
        unsuccessful {
            retry(3) {
                slackSend(channel: "#tradingcore-notify", color: "danger", message: ":red_circle: ${SLACK_MESSAGE} (${currentBuild.durationString.minus(' and counting')})")
            }
        }
        always {
            retry(3) {
                sh label: 'Clean docker images', script: '''#!/bin/bash -e
                    [ -z "$(docker images -q "${DOCKER_IMAGE_VEGA_CORE_LOCAL}")" ] || docker rmi "${DOCKER_IMAGE_VEGA_CORE_LOCAL}"
                    [ -z "$(docker images -q "${DOCKER_IMAGE_DATA_NODE_LOCAL}")" ] || docker rmi "${DOCKER_IMAGE_DATA_NODE_LOCAL}"
                '''
            }
        }
    }
}
