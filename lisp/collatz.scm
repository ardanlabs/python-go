(define collatz-step
  (lambda (n)
    (if (eq? (% n 2) 0)
	(/ n 2)
	(+ (* n 3) 1))))


(println (collatz-step 7)) ; 22
