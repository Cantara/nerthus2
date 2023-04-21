def vers
def outFile
def release = false
pipeline {
    agent any
    tools {
        go 'Go 1.20'
        maven 'Mvn'
    }
    environment {
        NEXUS_CREDS = credentials('Cantara-NEXUS')
    }
    stages {
        stage("pre") {
            steps {
                script {
                    if (env.TAG_NAME) {
                        vers = "${env.TAG_NAME}"
                        release = true
                    } else {
                        vers = "${env.GIT_COMMIT}"
                    }
                    artifactId = "nerthus2"
                    outFile = "${artifactId}-${vers}"
                    echo "New file: ${outFile}"
                }
            }
        }
        stage("test") {
            steps {
                script {
                    testApp()
                }
            }
        }
        stage("build") {
            steps {
                script {
                    echo "V: ${vers}"
                    echo "File: ${outFile}"
                    buildApp(outFile, vers)
                }
            }
        }
        stage("deploy") {
            steps {
                script {
                    echo 'deplying the application...'
                    echo "deploying version ${vers}"
                    if (release) {
                        sh "find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/${artifactId}/${vers}/{}  \\;"
                        sh "cd probe && find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/${artifactId}/${vers}/probe/{}  \\;"
                        sh "cd probe/health/cmd && find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/releases/no/cantara/gotools/${artifactId}/${vers}/probe/health/{}  \\;"
                        //sh "docker pull cantara/nerthus:${vers}"
                        //sh "docker image tag cantara/nerthus:${vers} cantara/nerthus:latest"
                        //sh "docker push cantara/nerthus:latest"
                    } else {
                        sh "find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/snapshots/no/cantara/gotools/${artifactId}/${vers}/{}  \\;"
                        sh "cd probe && find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/snapshots/no/cantara/gotools/${artifactId}/${vers}/probe/{}  \\;"
                        sh "cd probe/health/cmd && find . -name '${outFile}-*' -type f -exec curl -v -u "+'$NEXUS_CREDS'+" --upload-file {} https://mvnrepo.cantara.no/content/repositories/snapshots/no/cantara/gotools/${artifactId}/${vers}/probe/health/{}  \\;"
                        //sh "docker pull cantara/nerthus:${vers}"
                        //sh "docker image tag cantara/nerthus:${vers} cantara/nerthus:snapshot-latest"
                        //sh "docker push cantara/nerthus:snapshot-latest"
                    }
                    sh "rm ${outFile}-*"
                    sh "cd probe && rm ${outFile}-*"
                }
            }
        }
    }
}

def testApp() {
    echo 'testing the application...'
    sh './testRecursive.sh'
}

def buildApp(outFile, vers) {
    echo 'building the application...'
    buildFlags = "-X 'github.com/cantara/gober/webserver/health.Version=${vers}' -X 'github.com/cantara/gober/webserver/health.BuildTime=\$(date)' -X 'github.com/cantara/gober/webserver.Name=${artifactId}' "
    sh "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags \"${buildFlags}\" -o ${outFile}-linux-amd64"
    sh "CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags \"${buildFlags}\" -o ${outFile}-linux-arm64"
    sh "CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags \"${buildFlags}\" -o ${outFile}-darwin-amd64"
    sh "CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags \"${buildFlags}\" -o ${outFile}-darwin-arm64"
    //sh "docker buildx build --platform=linux/amd64,linux/arm64 --build-arg BUILDFLAGS=\"${buildFlags}\" -t cantara/nerthus:${vers} --push ."

    buildFlags = "-X 'github.com/cantara/gober/webserver/health.Version=${vers}' -X 'github.com/cantara/gober/webserver/health.BuildTime=\$(date)' -X 'github.com/cantara/gober/webserver.Name=${artifactId}-probe' "
    sh "cd probe && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags \"${buildFlags}\" -o ${outFile}-linux-amd64"
    sh "cd probe && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags \"${buildFlags}\" -o ${outFile}-linux-arm64"
    sh "cd probe && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags \"${buildFlags}\" -o ${outFile}-darwin-amd64"
    sh "cd probe && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags \"${buildFlags}\" -o ${outFile}-darwin-arm64"

    sh "cd probe/health/cmd && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${outFile}-linux-amd64"
    sh "cd probe/health/cmd && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${outFile}-linux-arm64"
    sh "cd probe/health/cmd && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${outFile}-darwin-amd64"
    sh "cd probe/health/cmd && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ${outFile}-darwin-arm64"
}
