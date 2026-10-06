#!/usr/bin/env python3
"""Seed 15 classic stdin/stdout Python contest problems via the admin API."""

import json
import urllib.request

BASE = "http://localhost:8759"


def req(method, path, data=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    body = None if data is None else json.dumps(data).encode()
    r = urllib.request.Request(BASE + path, data=body, headers=headers, method=method)
    with urllib.request.urlopen(r) as resp:
        return json.load(resp)


PROBLEMS = [
    {
        "slug": "a-plus-b",
        "title": "A + B",
        "difficulty": "easy",
        "order_index": 1,
        "statement_md": """## A + B

Read two integers **A** and **B** and print their sum.

### Input
A single line with two integers `A` and `B` (−10⁹ ≤ A, B ≤ 10⁹).

### Output
Print one integer — `A + B`.

### Example
```
Input:
3 5

Output:
8
```
""",
        "starter_code": "import sys\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    # TODO: print a + b\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    print(a + b)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "3 5\n", "expected_output": "8\n", "is_sample": True, "points": 0},
            {"input": "0 0\n", "expected_output": "0\n", "is_sample": True, "points": 0},
            {"input": "-2 7\n", "expected_output": "5\n", "is_sample": False, "points": 5},
            {"input": "1000000000 1000000000\n", "expected_output": "2000000000\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "maximum-of-two",
        "title": "Maximum of Two",
        "difficulty": "easy",
        "order_index": 2,
        "statement_md": """## Maximum of Two

Given two integers, print the larger one. If equal, print either.

### Input
Two integers on one line.

### Output
The maximum value.

### Example
```
Input:
4 9

Output:
9
```
""",
        "starter_code": "import sys\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    # TODO: print the larger of a and b\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    print(max(a, b))\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "4 9\n", "expected_output": "9\n", "is_sample": True, "points": 0},
            {"input": "7 7\n", "expected_output": "7\n", "is_sample": True, "points": 0},
            {"input": "-10 -3\n", "expected_output": "-3\n", "is_sample": False, "points": 5},
            {"input": "100 -100\n", "expected_output": "100\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "even-or-odd",
        "title": "Even or Odd",
        "difficulty": "easy",
        "order_index": 3,
        "statement_md": """## Even or Odd

Given an integer **N**, print `Even` if it is divisible by 2, otherwise print `Odd`.

### Input
A single integer N (−10⁹ ≤ N ≤ 10⁹).

### Output
`Even` or `Odd`.

### Example
```
Input:
7

Output:
Odd
```
""",
        "starter_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    # TODO: print \"Even\" or \"Odd\"\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    print('Even' if n % 2 == 0 else 'Odd')\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "7\n", "expected_output": "Odd\n", "is_sample": True, "points": 0},
            {"input": "4\n", "expected_output": "Even\n", "is_sample": True, "points": 0},
            {"input": "0\n", "expected_output": "Even\n", "is_sample": False, "points": 5},
            {"input": "-5\n", "expected_output": "Odd\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "factorial",
        "title": "Factorial",
        "difficulty": "easy",
        "order_index": 4,
        "statement_md": """## Factorial

Compute **N!** (1 × 2 × … × N). By definition, `0! = 1`.

### Input
A single integer N (0 ≤ N ≤ 12).

### Output
Print N!.

### Example
```
Input:
5

Output:
120
```
""",
        "starter_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    # TODO: print n!\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    ans = 1\n    for i in range(2, n + 1):\n        ans *= i\n    print(ans)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "5\n", "expected_output": "120\n", "is_sample": True, "points": 0},
            {"input": "0\n", "expected_output": "1\n", "is_sample": True, "points": 0},
            {"input": "1\n", "expected_output": "1\n", "is_sample": False, "points": 4},
            {"input": "10\n", "expected_output": "3628800\n", "is_sample": False, "points": 6},
        ],
    },
    {
        "slug": "sum-1-to-n",
        "title": "Sum from 1 to N",
        "difficulty": "easy",
        "order_index": 5,
        "statement_md": """## Sum from 1 to N

Print the sum of integers from 1 to **N** inclusive.

### Input
A single integer N (1 ≤ N ≤ 10⁹).

### Output
The sum `1 + 2 + … + N`.

### Example
```
Input:
5

Output:
15
```

Hint: use the formula `N * (N + 1) // 2` to stay fast for large N.
""",
        "starter_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    # TODO: print 1 + 2 + ... + n\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    print(n * (n + 1) // 2)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "5\n", "expected_output": "15\n", "is_sample": True, "points": 0},
            {"input": "1\n", "expected_output": "1\n", "is_sample": True, "points": 0},
            {"input": "100\n", "expected_output": "5050\n", "is_sample": False, "points": 5},
            {"input": "1000000000\n", "expected_output": "500000000500000000\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "reverse-string",
        "title": "Reverse a String",
        "difficulty": "easy",
        "order_index": 6,
        "statement_md": """## Reverse a String

Read a string and print it reversed.

### Input
One line containing the string S (1 ≤ |S| ≤ 10⁵). May contain letters, digits, and spaces (trimmed ends only if present as part of input line without trailing newline concerns — reverse the characters of the line excluding the final newline).

### Output
The reversed string.

### Example
```
Input:
hello

Output:
olleh
```
""",
        "starter_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().rstrip('\\n')\n    # TODO: print s reversed\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().rstrip('\\n')\n    print(s[::-1])\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "hello\n", "expected_output": "olleh\n", "is_sample": True, "points": 0},
            {"input": "a\n", "expected_output": "a\n", "is_sample": True, "points": 0},
            {"input": "Python3\n", "expected_output": "3nohtyP\n", "is_sample": False, "points": 5},
            {"input": "race car\n", "expected_output": "rac ecar\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "palindrome-check",
        "title": "Palindrome Check",
        "difficulty": "easy",
        "order_index": 7,
        "statement_md": """## Palindrome Check

A string is a palindrome if it reads the same forwards and backwards.  
Given a lowercase string, print `YES` if it is a palindrome, otherwise `NO`.

### Input
One string S (1 ≤ |S| ≤ 10⁵), lowercase letters only.

### Output
`YES` or `NO`.

### Example
```
Input:
level

Output:
YES
```
""",
        "starter_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().strip()\n    # TODO: print \"YES\" if palindrome else \"NO\"\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().strip()\n    print('YES' if s == s[::-1] else 'NO')\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "level\n", "expected_output": "YES\n", "is_sample": True, "points": 0},
            {"input": "abc\n", "expected_output": "NO\n", "is_sample": True, "points": 0},
            {"input": "a\n", "expected_output": "YES\n", "is_sample": False, "points": 3},
            {"input": "abba\n", "expected_output": "YES\n", "is_sample": False, "points": 4},
            {"input": "abcdcba\n", "expected_output": "YES\n", "is_sample": False, "points": 3},
        ],
    },
    {
        "slug": "count-vowels",
        "title": "Count Vowels",
        "difficulty": "easy",
        "order_index": 8,
        "statement_md": """## Count Vowels

Count how many vowels appear in the given string. Vowels are `a e i o u` (lowercase). The input is lowercase.

### Input
One lowercase string S (1 ≤ |S| ≤ 10⁵).

### Output
A single integer — the number of vowels.

### Example
```
Input:
education

Output:
5
```
""",
        "starter_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().strip()\n    # TODO: print count of vowels aeiou\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    s = sys.stdin.readline().strip()\n    print(sum(1 for c in s if c in 'aeiou'))\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "education\n", "expected_output": "5\n", "is_sample": True, "points": 0},
            {"input": "xyz\n", "expected_output": "0\n", "is_sample": True, "points": 0},
            {"input": "aeiou\n", "expected_output": "5\n", "is_sample": False, "points": 5},
            {"input": "programming\n", "expected_output": "3\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "fizzbuzz",
        "title": "FizzBuzz",
        "difficulty": "medium",
        "order_index": 9,
        "statement_md": """## FizzBuzz

For each integer from 1 to **N**:
- print `FizzBuzz` if divisible by both 3 and 5
- print `Fizz` if divisible by 3 only
- print `Buzz` if divisible by 5 only
- otherwise print the number itself

Print each value on its own line.

### Input
A single integer N (1 ≤ N ≤ 1000).

### Output
N lines as described.

### Example
```
Input:
5

Output:
1
2
Fizz
4
Buzz
```
""",
        "starter_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    # TODO: for i in 1..n print FizzBuzz / Fizz / Buzz / i\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    for i in range(1, n + 1):\n        if i % 15 == 0:\n            print('FizzBuzz')\n        elif i % 3 == 0:\n            print('Fizz')\n        elif i % 5 == 0:\n            print('Buzz')\n        else:\n            print(i)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "5\n", "expected_output": "1\n2\nFizz\n4\nBuzz\n", "is_sample": True, "points": 0},
            {"input": "1\n", "expected_output": "1\n", "is_sample": True, "points": 0},
            {"input": "15\n", "expected_output": "1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz\n", "is_sample": False, "points": 10},
        ],
    },
    {
        "slug": "two-sum-sorted",
        "title": "Two Sum (Sorted)",
        "difficulty": "medium",
        "order_index": 10,
        "statement_md": """## Two Sum (Sorted)

You are given a **sorted** array of distinct integers and a target.  
Find two indices (1-based) of numbers that add up to the target. There is exactly one solution.

### Input
- Line 1: integer N (2 ≤ N ≤ 10⁵)
- Line 2: N strictly increasing integers
- Line 3: target integer T

### Output
Two 1-based indices `i j` (i < j) such that `A[i] + A[j] = T`.

### Example
```
Input:
4
2 7 11 15
9

Output:
1 2
```
""",
        "starter_code": "import sys\n\ndef main():\n    data = sys.stdin.read().strip().split()\n    n = int(data[0])\n    a = list(map(int, data[1:n + 1]))\n    t = int(data[n + 1])\n    # TODO: print 1-based indices i j with a[i-1] + a[j-1] == t\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    data = sys.stdin.read().strip().split()\n    n = int(data[0])\n    a = list(map(int, data[1:n+1]))\n    t = int(data[n+1])\n    i, j = 0, n - 1\n    while i < j:\n        s = a[i] + a[j]\n        if s == t:\n            print(i + 1, j + 1)\n            return\n        if s < t:\n            i += 1\n        else:\n            j -= 1\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "4\n2 7 11 15\n9\n", "expected_output": "1 2\n", "is_sample": True, "points": 0},
            {"input": "3\n1 2 3\n5\n", "expected_output": "2 3\n", "is_sample": True, "points": 0},
            {"input": "5\n-5 -2 0 3 8\n3\n", "expected_output": "2 5\n", "is_sample": False, "points": 5},
            {"input": "2\n1 100\n101\n", "expected_output": "1 2\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "binary-search",
        "title": "Binary Search",
        "difficulty": "medium",
        "order_index": 11,
        "statement_md": """## Binary Search

Given a sorted array and a query value X, print the **1-based index** of X if present, otherwise print `-1`.

### Input
- Line 1: N (1 ≤ N ≤ 10⁵)
- Line 2: N sorted ascending integers
- Line 3: X

### Output
Index of X (1-based) or `-1`.

### Example
```
Input:
5
1 3 5 7 9
7

Output:
4
```
""",
        "starter_code": "import sys\n\ndef main():\n    data = sys.stdin.read().strip().split()\n    n = int(data[0])\n    a = list(map(int, data[1:n + 1]))\n    x = int(data[n + 1])\n    # TODO: print 1-based index of x in a, or -1\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    data = sys.stdin.read().strip().split()\n    n = int(data[0])\n    a = list(map(int, data[1:n+1]))\n    x = int(data[n+1])\n    lo, hi = 0, n - 1\n    while lo <= hi:\n        mid = (lo + hi) // 2\n        if a[mid] == x:\n            print(mid + 1)\n            return\n        if a[mid] < x:\n            lo = mid + 1\n        else:\n            hi = mid - 1\n    print(-1)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "5\n1 3 5 7 9\n7\n", "expected_output": "4\n", "is_sample": True, "points": 0},
            {"input": "5\n1 3 5 7 9\n2\n", "expected_output": "-1\n", "is_sample": True, "points": 0},
            {"input": "1\n42\n42\n", "expected_output": "1\n", "is_sample": False, "points": 4},
            {"input": "6\n-10 -3 0 4 8 20\n-3\n", "expected_output": "2\n", "is_sample": False, "points": 6},
        ],
    },
    {
        "slug": "gcd",
        "title": "GCD",
        "difficulty": "medium",
        "order_index": 12,
        "statement_md": """## Greatest Common Divisor

Given two positive integers A and B, print `gcd(A, B)`.

### Input
Two integers A and B (1 ≤ A, B ≤ 10¹²).

### Output
Their GCD.

### Example
```
Input:
12 18

Output:
6
```
""",
        "starter_code": "import sys\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    # TODO: print gcd(a, b)\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\nimport math\n\ndef main():\n    a, b = map(int, sys.stdin.read().split())\n    print(math.gcd(a, b))\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "12 18\n", "expected_output": "6\n", "is_sample": True, "points": 0},
            {"input": "7 13\n", "expected_output": "1\n", "is_sample": True, "points": 0},
            {"input": "100 10\n", "expected_output": "10\n", "is_sample": False, "points": 4},
            {"input": "123456789012 987654321098\n", "expected_output": "2\n", "is_sample": False, "points": 6},
        ],
    },
    {
        "slug": "prime-check",
        "title": "Prime Check",
        "difficulty": "medium",
        "order_index": 13,
        "statement_md": """## Prime Check

Determine whether **N** is a prime number. Print `YES` if prime, otherwise `NO`.

### Input
A single integer N (1 ≤ N ≤ 10¹²).

### Output
`YES` or `NO`.

### Example
```
Input:
17

Output:
YES
```
""",
        "starter_code": "import sys\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    # TODO: print \"YES\" if n is prime else \"NO\"\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef is_prime(n):\n    if n < 2:\n        return False\n    if n % 2 == 0:\n        return n == 2\n    i = 3\n    while i * i <= n:\n        if n % i == 0:\n            return False\n        i += 2\n    return True\n\ndef main():\n    n = int(sys.stdin.read().strip())\n    print('YES' if is_prime(n) else 'NO')\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "17\n", "expected_output": "YES\n", "is_sample": True, "points": 0},
            {"input": "1\n", "expected_output": "NO\n", "is_sample": True, "points": 0},
            {"input": "2\n", "expected_output": "YES\n", "is_sample": False, "points": 3},
            {"input": "1000000007\n", "expected_output": "YES\n", "is_sample": False, "points": 4},
            {"input": "91\n", "expected_output": "NO\n", "is_sample": False, "points": 3},
        ],
    },
    {
        "slug": "matrix-diagonal-sum",
        "title": "Matrix Diagonal Sum",
        "difficulty": "medium",
        "order_index": 14,
        "statement_md": """## Matrix Diagonal Sum

Given an N×N matrix, print the sum of the **primary diagonal** (top-left to bottom-right).

### Input
- Line 1: N (1 ≤ N ≤ 200)
- Next N lines: N integers each

### Output
Sum of elements where row index equals column index (0-based).

### Example
```
Input:
3
1 2 3
4 5 6
7 8 9

Output:
15
```
""",
        "starter_code": "import sys\n\ndef main():\n    data = list(map(int, sys.stdin.read().split()))\n    n = data[0]\n    a = data[1:]\n    # matrix row i is a[i*n : (i+1)*n]\n    # TODO: print sum of primary diagonal a[i*n + i]\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    data = list(map(int, sys.stdin.read().split()))\n    n = data[0]\n    a = data[1:]\n    s = 0\n    for i in range(n):\n        s += a[i * n + i]\n    print(s)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "3\n1 2 3\n4 5 6\n7 8 9\n", "expected_output": "15\n", "is_sample": True, "points": 0},
            {"input": "1\n42\n", "expected_output": "42\n", "is_sample": True, "points": 0},
            {"input": "2\n1 0\n0 1\n", "expected_output": "2\n", "is_sample": False, "points": 5},
            {"input": "4\n1 2 3 4\n5 6 7 8\n9 10 11 12\n13 14 15 16\n", "expected_output": "34\n", "is_sample": False, "points": 5},
        ],
    },
    {
        "slug": "longest-word",
        "title": "Longest Word",
        "difficulty": "hard",
        "order_index": 15,
        "statement_md": """## Longest Word

Given a sentence, print the **longest word**. If there are ties, print the first longest word.  
Words are separated by spaces. Punctuation is not present.

### Input
One line with words separated by single spaces (1 ≤ length ≤ 10⁵).

### Output
The longest word.

### Example
```
Input:
I love programming contests

Output:
programming
```
""",
        "starter_code": "import sys\n\ndef main():\n    words = sys.stdin.readline().split()\n    # TODO: print the longest word (first on ties)\n    pass\n\nif __name__ == '__main__':\n    main()\n",
        "reference_code": "import sys\n\ndef main():\n    words = sys.stdin.readline().split()\n    best = ''\n    for w in words:\n        if len(w) > len(best):\n            best = w\n    print(best)\n\nif __name__ == '__main__':\n    main()\n",
        "cases": [
            {"input": "I love programming contests\n", "expected_output": "programming\n", "is_sample": True, "points": 0},
            {"input": "a bb ccc dd\n", "expected_output": "ccc\n", "is_sample": True, "points": 0},
            {"input": "same size word here\n", "expected_output": "same\n", "is_sample": False, "points": 5},
            {"input": "to be or not to be that is the question\n", "expected_output": "question\n", "is_sample": False, "points": 5},
        ],
    },
]


def main():
    login = req("POST", "/api/admin/login", {"username": "admin", "password": "admin123"})
    token = login["token"]

    existing = req("GET", "/api/admin/problems", token=token)
    for p in existing:
        req("DELETE", f"/api/admin/problems/{p['id']}", token=token)
        print(f"deleted old problem id={p['id']} ({p['title']})")

    for p in PROBLEMS:
        cases = p.pop("cases")
        created = req("POST", "/api/admin/problems", p, token=token)
        for i, c in enumerate(cases):
            c["order_index"] = i
        req("PUT", f"/api/admin/problems/{created['id']}/test-cases", {"test_cases": cases}, token=token)
        print(f"created [{created['id']}] {created['order_index']}. {created['title']} ({created['difficulty']}) — {len(cases)} cases")

    print(f"\nDone: {len(PROBLEMS)} problems seeded.")


if __name__ == "__main__":
    main()
