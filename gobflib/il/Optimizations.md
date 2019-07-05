# Compress


```go
dataadd(1)
dataadd(1)
// simplifies to
dataadd(2)
```

# Zero

```go
for data[datap] != 0 {
	dataadd(255)
}
// equates to
dataset0)
```


# Multiply

```go
datapadd(7)
for data[datap] != 0 {
	dataadd(255)
	datapadd(-7)
	dataadd(1)
	datapadd(7)
}
datapadd(-7)
// equates to
datapadd(7)
for data[datap] != 0 {
	dataadd(255)
	datapadd(-7)
	dataadd(1)
	datapadd(7)
}
datapadd(-7)
```

# Sample 2

```go
datapadd(-9)
for data[datap] != 0 {
	datapadd(-9)
}
datapadd(7)
for data[datap] != 0 {
	dataadd(255)
	datapadd(-7)
	dataadd(1)
	datapadd(7)
}
datapadd(-7)
for data[datap] != 0 {
	dataadd(255)
	datapadd(7)
	dataadd(1)
	datapadd(-2)
	dataadd(1)
	datapadd(-5)
}
datapadd(9)
```


######

```go
datapadd(9)
for data[datap] != 0 {
	datapadd(8)
	for data[datap] != 0 {
		dataadd(255)
		datapadd(-7)
		dataadd(1)
		datapadd(7)
	}
	datapadd(-7)
	for data[datap] != 0 {
		dataadd(255)
		datapadd(7)
		dataadd(1)
		datapadd(-2)
		dataadd(1)
		datapadd(-3)
		dataadd(1)
		datapadd(-2)
	}
	datapadd(8)
}
datapadd(-9)
```

######

```go
dataadd(1)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
datapadd(1)
dataset(0)
```