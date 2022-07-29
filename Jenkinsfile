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
        string( name: 'PROTOS_BRANCH', defaultValue: 'develop',
                description: 'Git branch, tag or hash of the vegaprotocol/protos repository')
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
    }
    environment {
        CGO_ENABLED = 0
        GO111MODULE = 'on'
        BUILD_UID="${BUILD_NUMBER}-${EXECUTOR_NUMBER}"
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
                    sh '''#!/bin/bash -e
                        go mod download -x
                    '''
                }
            }
        }

        //
        // Begin COMPILE
        //
        stage('Compile') {
            failFast true
            matrix {
                axes {
                    axis {
                        name 'GOOS'
                        values 'linux', 'darwin', 'windows'
                    }
                    axis {
                        name 'GOARCH'
                        values 'amd64', 'arm64'
                    }
                }
                excludes {
                    exclude {
                        axis {
                            name 'GOOS'
                            values 'windows'
                        }
                        axis {
                            name 'GOARCH'
                            values 'arm64'
                        }
                    }
                }
                stages {
                    stage('Build') {
                        environment {
                            GOOS         = "${GOOS}"
                            GOARCH       = "${GOARCH}"
                        }
                        options { retry(3) }
                        steps {
                            sh 'printenv'
                            dir('vega') {
                                sh label: 'Compile', script: """#!/bin/bash -e
                                    go build -v \
                                        -o ../build-${GOOS}-${GOARCH}/ \
                                        ./cmd/vega \
                                        ./cmd/data-node \
                                        ./cmd/vegawallet
                                """
                                sh label: 'check for modifications', script: 'git diff'
                            }
                            dir("build-${GOOS}-${GOARCH}") {
                                sh label: 'list files', script: '''#!/bin/bash -e
                                    pwd
                                    ls -lah
                                '''
                                sh label: 'Sanity check', script: '''#!/bin/bash -e
                                    file *
                                '''
                                script {
                                    if ( GOOS == "linux" && GOARCH == "amd64" ) {
                                        sh label: 'get version', script: '''#!/bin/bash -e
                                            ./vega version
                                            ./data-node version
                                            ./vegawallet version
                                        '''
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        //
        // End COMPILE
        //

        //
        // Begin LINTERS
        //
        // stage('Linters') {
        //     parallel {
        //         stage('linters') {
        //             steps {
        //                 dir('vega') {
        //                     sh '''#!/bin/bash -e
        //                         golangci-lint run -v --config .golangci.toml
        //                     '''
        //                 }
        //             }
        //         }
        //         stage('shellcheck') {
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     sh "git ls-files '*.sh'"
        //                     sh "git ls-files '*.sh' | xargs shellcheck"
        //                 }
        //             }
        //         }
        //         stage('yamllint') {
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     sh "git ls-files '*.yml' '*.yaml'"
        //                     sh "git ls-files '*.yml' '*.yaml' | xargs yamllint -s -d '{extends: default, rules: {line-length: {max: 160}}}'"
        //                 }
        //             }
        //         }
        //         stage('json format') {
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     sh "git ls-files '*.json'"
        //                     sh "for f in \$(git ls-files '*.json'); do echo \"check \$f\"; jq empty \"\$f\"; done"
        //                 }
        //             }
        //         }
        //         stage('markdown spellcheck') {
        //             environment {
        //                 FORCE_COLOR = '1'
        //             }
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     ansiColor('xterm') {
        //                         sh 'mdspell --en-gb --ignore-acronyms --ignore-numbers --no-suggestions --report "*.md" "docs/**/*.md"'
        //                     }
        //                 }
        //             }
        //         }
        //         stage('approbation') {
        //             when {
        //                 anyOf {
        //                     branch 'develop'
        //                     branch 'main'
        //                     branch 'master'
        //                 }
        //             }
        //             steps {
        //                 script {
        //                     runApprobation ignoreFailure: !isPRBuild(),
        //                         vegaVersion: commitHash
        //                 }
        //             }
        //         }
        //     }
        // }
        //
        // End LINTERS
        //

        //
        // Begin TESTS
        //
        // stage('Tests') {
        //     parallel {
        //         stage('unit tests') {
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     sh 'go test -v ./... 2>&1 | tee unit-test-results.txt && cat unit-test-results.txt | go-junit-report > vega-unit-test-report.xml'
        //                     junit checksName: 'Unit Tests', testResults: 'vega-unit-test-report.xml'
        //                 }
        //             }
        //         }
        //         stage('unit tests with race') {
        //             environment {
        //                 CGO_ENABLED = 1
        //             }
        //             options { retry(3) }
        //             steps {
        //                 dir('vega') {
        //                     sh 'go test -v -race ./... 2>&1 | tee unit-test-race-results.txt && cat unit-test-race-results.txt | go-junit-report > vega-unit-test-race-report.xml'
        //                     junit checksName: 'Unit Tests with Race', testResults: 'vega-unit-test-race-report.xml'
        //                 }
        //             }
        //         }
        //         stage('vega/integration tests') {
        //             options { retry(3) }
        //             steps {
        //                 dir('vega/core/integration') {
        //                     sh 'godog build -o integration.test && ./integration.test --format=junit:vega-integration-report.xml'
        //                     junit checksName: 'Integration Tests', testResults: 'vega-integration-report.xml'
        //                 }
        //             }
        //         }
        //         stage('System Tests Network Smoke') {
        //             steps {
        //                 script {
        //                     systemTestsCapsule ignoreFailure: !isPRBuild(),
        //                         timeout: 30,
        //                         vegaVersion: commitHash,
        //                         protos: params.PROTOS_BRANCH,
        //                         systemTests: params.SYSTEM_TESTS_BRANCH,
        //                         vegacapsule: params.VEGACAPSULE_BRANCH,
        //                         vegatools: params.VEGATOOLS_BRANCH,
        //                         devopsInfra: params.DEVOPS_INFRA_BRANCH,
        //                         devopsScripts: params.DEVOPSSCRIPTS_BRANCH,
        //                         testMark: "network_infra_smoke"
        //                 }
        //             }
        //         }
        //         stage('Capsule System Tests') {
        //             steps {
        //                 script {
        //                     systemTestsCapsule ignoreFailure: !isPRBuild(),
        //                         timeout: 30,
        //                         vegaVersion: commitHash,
        //                         protos: params.PROTOS_BRANCH,
        //                         systemTests: params.SYSTEM_TESTS_BRANCH,
        //                         vegacapsule: params.VEGACAPSULE_BRANCH,
        //                         vegatools: params.VEGATOOLS_BRANCH,
        //                         devopsInfra: params.DEVOPS_INFRA_BRANCH,
        //                         devopsScripts: params.DEVOPSSCRIPTS_BRANCH
        //                 }
        //             }
        //         }
        //     }
        // }
        //
        // End TESTS
        //

        //
        // Begin PUBLISH
        //
        stage('Publish') {
            environment {
                DOCKER_PUBLISH = "${ isPRBuild() ? ' ' : '--publish' }"
                DOCKER_BUILD_ARCH = "${ isPRBuild() ? 'linux/amd64' : 'linux/arm64,linux/amd64' }"
                DOCKER_IMAGE_TAG = "${ env.TAG_NAME ? env.TAG_NAME : env.BRANCH_NAME }"
            }
            parallel {
                stage('vega docker image') {
                    environment {
                        BUILDX_BUILDER_NAME="vega-${BUILD_UID}"
                    }
                    stages {
                        stage('Create builder') {
                            steps {
                                sh label: 'create buildx builder', script: """#!/bin/bash -e
                                    docker buildx create --name ${BUILDX_BUILDER_NAME}
                                """
                            }
                        }
                        stage('docker build') {
                            steps {
                                dir('vega') {
                                    sh 'printenv'
                                    sh label: 'build and publish multi-arch docker image', script: """#!/bin/bash -e
                                        docker buildx build \
                                            --builder ${BUILDX_BUILDER_NAME} \
                                            --platform=${DOCKER_BUILD_ARCH} \
                                            -f docker/vega.dockerfile \
                                            -t ghcr.io/vegaprotocol/vega/vega:${DOCKER_IMAGE_TAG} \
                                            ${DOCKER_PUBLISH} \
                                            .
                                    """
                                }
                            }
                            post {
                                always {
                                    sh label: 'destroy buildx builder', script: """#!/bin/bash -e
                                        docker buildx rm --force ${BUILDX_BUILDER_NAME}
                                    """
                                }
                            }
                        }
                    }
                }
                stage('data-node docker image') {
                    environment {
                        BUILDX_BUILDER_NAME="data-node-${BUILD_UID}"
                    }
                    stages {
                        stage('Create builder') {
                            steps {
                                sh label: 'create buildx builder', script: """#!/bin/bash -e
                                    docker buildx create --name ${BUILDX_BUILDER_NAME}
                                """
                            }
                        }
                        stage('docker build') {
                            steps {
                                dir('vega') {
                                    sh label: 'build and publish multi-arch docker image', script: """#!/bin/bash -e
                                        docker buildx build \
                                            --builder ${BUILDX_BUILDER_NAME} \
                                            --platform=${DOCKER_BUILD_ARCH} \
                                            -f docker/data-node.dockerfile \
                                            -t ghcr.io/vegaprotocol/vega/data-node:${DOCKER_IMAGE_TAG} \
                                            ${DOCKER_PUBLISH} \
                                            .
                                    """
                                }
                            }
                            post {
                                always {
                                    sh label: 'destroy buildx builder', script: """#!/bin/bash -e
                                        docker buildx rm --force ${BUILDX_BUILDER_NAME}
                                    """
                                }
                            }
                        }
                    }
                }
                stage('vegawallet docker image') {
                    environment {
                        BUILDX_BUILDER_NAME="vegawallet-${BUILD_UID}"
                    }
                    stages {
                        stage('Create builder') {
                            steps {
                                sh label: 'create buildx builder', script: """#!/bin/bash -e
                                    docker buildx create --name ${BUILDX_BUILDER_NAME}
                                """
                            }
                        }
                        stage('docker build') {
                            steps {
                                dir('vega') {
                                    sh label: 'build and publish multi-arch docker image', script: """#!/bin/bash -e
                                        docker buildx build \
                                            --builder ${BUILDX_BUILDER_NAME} \
                                            --platform=${DOCKER_BUILD_ARCH} \
                                            -f docker/vegawallet.dockerfile \
                                            -t ghcr.io/vegaprotocol/vega/vegawallet:${DOCKER_IMAGE_TAG} \
                                            ${DOCKER_PUBLISH} \
                                            .
                                    """
                                }
                            }
                            post {
                                always {
                                    sh label: 'destroy buildx builder', script: """#!/bin/bash -e
                                        docker buildx rm --force ${BUILDX_BUILDER_NAME}
                                    """
                                }
                            }
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


                stage('release to GitHub') {
                    when {
                        buildingTag()
                    }
                    environment {
                        RELEASE_URL = "https://github.com/vegaprotocol/vega/releases/tag/${TAG_NAME}"
                    }
                    options { retry(3) }
                    steps {
                        sh label: 'copy artefacts to publish to one directory', script: '''#!/bin/bash -e
                            mkdir release
                            # linux
                            cp ./build-linux-amd64/vega ./release/vega-linux-amd64
                            cp ./build-linux-amd64/data-node ./release/data-node-linux-amd64
                            cp ./build-linux-arm64/vega ./release/vega-linux-arm64
                            cp ./build-linux-arm64/data-node ./release/data-node-linux-arm64
                            # MacOS
                            cp ./build-darwin-amd64/vega ./release/vega-macos-amd64
                            cp ./build-darwin-amd64/data-node ./release/data-node-macos-amd64
                            cp ./build-darwin-arm64/vega ./release/vega-macos-arm64
                            cp ./build-darwin-arm64/data-node ./release/data-node-macos-arm64
                            # Windows
                            cp ./build-windows-amd64/vega ./release/vega-windows-amd64
                            cp ./build-windows-amd64/data-node ./release/data-node-windows-amd64
                        '''
                        dir('release') {
                            script {
                                withGHCLI('credentialsId': 'github-vega-ci-bot-artifacts') {
                                    sh label: 'Upload artifacts', script: '''#!/bin/bash -e
                                        [[ $TAG_NAME =~ '-pre' ]] && prerelease='--prerelease' || prerelease=''

                                        gh release view $TAG_NAME && gh release upload $TAG_NAME ./* \
                                            || gh release create $TAG_NAME $prerelease ./*
                                    '''
                                }
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
                devnetDeploy vegaCore: commitHash,
                    wait: false
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
    }
}
