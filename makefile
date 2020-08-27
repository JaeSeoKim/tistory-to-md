CC = go build
CFLAGS = -v

VERSION = 0.4

TARGET = tistory-to-md-$(VERSION)_x64

# env Setup
ENVWINDOWS64 = GOOS=windows GOARCH=amd64
ENVLINUX64 = GOOS=linux GOARCH=amd64

all : windows linux

re : clean all

windows :
	env $(ENVWINDOWS64) $(CC) -o $(TARGET).exe $(CFLAGS) main.go

linux :
	env $(ENVLINUX64) $(CC) -o $(TARGET) -v main.go

releas : all
	zip $(TARGET).zip ./template/*  $(TARGET).exe  $(TARGET)

fclean :
	rm -rf $(TARGET).exe $(TARGET) $(TARGET).zip

clean : fclean
