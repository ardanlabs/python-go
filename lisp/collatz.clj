(def collatz-step
  (fn [n]
    (if (= (mod n 2) 0)
	(/ n 2)
	(+ (* n 3) 1))))

(println (collatz-step 7)) ; 22
