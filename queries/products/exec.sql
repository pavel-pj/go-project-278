-- name: CreateProduct :one
INSERT INTO products (slug,title,description,price_cents)
values($1,$2,$3,$4)
RETURNING id,slug,title,description,price_cents,created_at;

-- name: DeleteAllProducts :exec
Delete from products;

-- name: UpdateProductPrice :execrows
UPDATE products set price_cents=$1 where id =$2;
 
-- name: DeleteProduct :execrows
DELETE from products where id = $1; 