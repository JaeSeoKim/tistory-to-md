# tistory-to-md

[![GO Version][go-image]][go-image]

> Tistory의 OpenAPI를 사용하여 게시글과 이미지를 MarkDown으로 백업하는 프로젝트 입니다.

## Installation

(Linux & Windows) & ARCH=amd64

[releases](https://github.com/JaeSeoKim/tistory-to-md/releases/) 를 다운 받아서 사용 가능합니다.

OS X & Linux & Windows

```sh
go build main.go
```

## Usage example

프로그램 실행후 아래와 같은 화면이 보이게 되면 설명하는 URL에 들어가 Tokken를 확인 후 터미널에 입력 합니다.

```bash
INFO: http://localhost:8080/를 통하여 서버가 실행 되었습니다.
*중요*
https://www.tistory.com/oauth/authorize?client_id=c7a92e9bd27df7b405ea3678e03eb460&redirect_uri=http://localhost:8080/&response_type=token
URL로 들어가 Tistory 인증후 화면에 나오는 AccessToken를 입력해주세요!
```
실행 후 `./result/{username}/{postTitle}.md` `./result/{username}/image/{postTitle}/*` 경로에 파일이 백업 된 것을 확인 할 수 있습니다.

개발자의 블로그에서만 테스트를 하여서 오류가 많습니다!
오류 발견시 `Issue` 남겨주세요

## Release History

* 0.2
    * 모든 작업 종료 후 Terminal 대기 추가.
* 0.1
    * 최초 릴리즈

<!-- Markdown link & img dfn's -->
[go-image]: https://img.shields.io/github/go-mod/go-version/JaeSeoKim/tistory-to-md?filename=go.mod
