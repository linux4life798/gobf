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
data[datap] = 0
```


# Add

```go
// [ v1 | _ | v2 ]
datapadd(2)
for data[datap] != 0 {
	dataadd(255)

	datapadd(-2)
	dataadd(1)
	datapadd(2)
}
datapadd(-2)

// equates to
// [ v1+v2 | _ | 0 ]
data[datap] += data[datap+2]
data[datap+2] = 0
```

```go
for data[datap] != 0 {
	dataaddvector([]byte{0xff, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1})
}

// or

for data[datap] != 0 {
	datapadd(-6)
	dataaddvector([]byte{0x1, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff})
	datapadd(6)
}

// transforms into
// determine last datap offset to determine loop invariant

func dataaddlinvector(vec []byte, multiplier byte) {
	for i := range vec {
		data[datap+i] += vec[i] * multiplier
	}
}

dataaddlinvector([]byte{0x1, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff}, data[datap])
data[datap] = 0
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