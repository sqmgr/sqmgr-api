# pwned - A Go client to check for pwned passwords

Uses [Have I Been Pwned API](https://haveibeenpwned.com/API/v2#PwnedPasswords) to see if a password has been pwned.

## Usage

```
pwnedCount, err := pwned.Count(password)
if err != nil {
	log.Fatal("could check pwnage: %v", err)
}
fmt.Printf("That password has been pwned %d times\n", pwnedCount)
```
