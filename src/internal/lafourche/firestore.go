package lafourche

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Accès REST à Firestore (panier du compte). Le panier de La Fourche est stocké
// dans Firestore et synchronisé entre mobile et web :
//   customers/<uid>.shoppingCartId  ->  carts/<shoppingCartId> = { SKU: quantité }

// fsValue représente une valeur de champ Firestore (forme REST).
type fsValue struct {
	StringValue  *string `json:"stringValue,omitempty"`
	IntegerValue *string `json:"integerValue,omitempty"`
}

func (c *Client) firestoreURL(docPath string) string {
	return fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents/%s",
		c.cfg.FirebaseProjectID, docPath)
}

// firestoreGet récupère les champs d'un document. Un 404 renvoie (nil, 404, nil).
func (c *Client) firestoreGet(ctx context.Context, docPath string) (map[string]fsValue, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.firestoreURL(docPath), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.session.AccessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, http.StatusNotFound, nil
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("firestore %s: http %d: %s", docPath, resp.StatusCode, string(raw))
	}
	var doc struct {
		Fields map[string]fsValue `json:"fields"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, resp.StatusCode, err
	}
	return doc.Fields, resp.StatusCode, nil
}

// firestorePatch applique un updateMask sur un document. Les champs absents de
// "fields" mais présents dans le mask sont supprimés.
func (c *Client) firestorePatch(ctx context.Context, docPath string, fieldPaths []string, fields map[string]fsValue) error {
	u := c.firestoreURL(docPath) + "?"
	for _, fp := range fieldPaths {
		// Les SKU contiennent des tirets -> chemin de champ entre backticks.
		u += "updateMask.fieldPaths=" + urlQueryEscape("`"+fp+"`") + "&"
	}
	body, err := json.Marshal(map[string]any{"fields": fields})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("firestore patch %s: http %d: %s", docPath, resp.StatusCode, string(raw))
	}
	return nil
}

func urlQueryEscape(s string) string {
	// Échappement minimal pour un query param Firestore (backticks + tirets).
	out := make([]byte, 0, len(s)*3)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			out = append(out, ch)
		} else {
			out = append(out, '%', "0123456789ABCDEF"[ch>>4], "0123456789ABCDEF"[ch&0xf])
		}
	}
	return string(out)
}
