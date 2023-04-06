CC = go
FLAGS = -i -o
NAME_ARM7 = ddnsclient_arm
NAME = ddnsclient

default:
	$(CC) build $(FLAGS) $(NAME) main.go

clean:
	rm $(NAME) $(NAME_ARM7)

arm:
	GOARM=7 GOARCH=arm $(CC) build -o $(NAME_ARM7)