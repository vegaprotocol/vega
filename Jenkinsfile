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
    agent {
        label params.NODE_LABEL
    }
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
        string( name: 'NODE_LABEL', defaultValue: 's-4vcpu-8gb',
                description: 'Label on which vega build should be run, if empty any any node is used')
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        BUILD_UID="${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
    }

    stages {
    	stage('CI Config') {
                steps {
                    sh "printenv"
                    echo "params=${params.inspect()}"
                    script {
                        publicIP = agent.getPublicIP()
                        print("Jenkins Agent public IP is: " + publicIP)
                    }
                }
            }

        stage('Config') {
            steps {
                cleanWs()
                sh 'printenv'
                echo "params=${params}"
                echo "isPRBuild=${isPRBuild()}"
                script {
                    params = pr.injectPRParams()
                    originRepo = pr.getOriginRepo('vegaprotocol/vega')
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

        stage('publish to vega-dev-releases') {
            when {
                branch 'develop'
            }
            steps {
                startVegaDevRelease vegaVersion: versionHash,
                    jenkinsSharedLib: params.JENKINS_SHARED_LIB_BRANCH
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
        //
        // End PREPARE
        //

        //
        // Begin LINTERS
        //
        stage('Linters') {
            parallel {
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
                                sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md" "!UPGRADING.md" "!DOCUMENTATION_STYLE.md"'
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
            }
        }
        //
        // End LINTERS
        //

        //
        // Begin TESTS
        //
        stage('Tests') {
            parallel {
                stage('unit tests') {
                    options { retry(3) }
                    steps {
                        dir('vega') {
                            sh 'go test -short -timeout 30m -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
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
                            sh 'go test -short -timeout 30m  -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
                            junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
                        }
                    }
                }
                stage('core/integration tests') {
                    options { retry(3) }
                    steps {
                        dir('vega/core/integration') {
                            sh 'godog build -o core_integration.test && ./core_integration.test --format=junit:core-integration-report.xml'
                            junit checksName: 'Core Integration Tests', testResults: 'core-integration-report.xml'
                        }
                    }
                }
                stage('datanode/integration tests') {
                    options { retry(3) }
                    steps {
                        dir('vega/datanode/integration') {
                            sh 'go test -v ./... 2>&1 | tee integration-test-results.txt && cat integration-test-results.txt | go-junit-report > datanode-integration-test-report.xml'
                            junit checksName: 'Datanode Integration Tests', testResults: 'datanode-integration-test-report.xml'
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
                stage('Vegavisor autoinstall and pup') {
                    steps {
                        build(
                            job: '/common/visor-autoinstall-and-pup',
                            propagate: true, // fast fail
                            wait: true,
                            parameters: [
                                string(name: 'RELEASES_REPO', value: 'vegaprotocol/vega-dev-releases-system-tests'),
                                string(name: 'VEGA_BRANCH', value: commitHash),
                                string(name: 'SYSTEM_TESTS_BRANCH', value: params.SYSTEM_TESTS_BRANCH ?: pipelineDefaults.capsuleSystemTests.branchSystemTests),
                                string(name: 'VEGATOOLS_BRANCH', value: params.VEGATOOLS_BRANCH ?: pipelineDefaults.capsuleSystemTests.branchVegatools),
                                string(name: 'VEGACAPSULE_BRANCH', value: params.VEGACAPSULE_BRANCH ?: pipelineDefaults.capsuleSystemTests.branchVegaCapsule),
                                string(name: 'DEVOPSSCRIPTS_BRANCH', value: params.DEVOPSSCRIPTS_BRANCH ?: pipelineDefaults.capsuleSystemTests.branchDevopsScripts),
                                booleanParam(name: 'CREATE_RELEASE', value: true),
                                string(name: 'JENKINS_SHARED_LIB_BRANCH', value: params.JENKINS_SHARED_LIB_BRANCH ?: pipelineDefaults.capsuleSystemTests.jenkinsSharedLib),
                            ]
                        )
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
                        stage('proto format check') {
                            options { retry(3) }
                            steps {
                                dir('vega') {
                                    sh '''#!/bin/bash -e
                                        make proto_format_check
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
                stage('Compile visor') {
                    options { retry(3) }
                    steps {
                        sh 'printenv'
                        dir('vega') {
                            sh label: 'Compile', script: """#!/bin/bash -e
                                go build -v \
                                    -o ../build/ \
                                    ./cmd/visor
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
                                ./visor version
                            '''
                        }
                    }
                }
            }
        }
        //
        // End TESTS
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
            cleanWs()
        }
    }
}
