"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useReducer,
} from "react";

export type CartItem = {
  productVariantId: string;
  productSlug: string;
  productName: string;
  variantName: string;
  price: number;
  quantity: number;
  stockQuantity: number;
};

type CartState = CartItem[];

type CartAction =
  | { type: "hydrate"; items: CartItem[] }
  | { type: "add"; item: CartItem }
  | { type: "updateQuantity"; productVariantId: string; quantity: number }
  | { type: "remove"; productVariantId: string }
  | { type: "clear" };

const STORAGE_KEY = "ecomwa_cart";

function cartReducer(state: CartState, action: CartAction): CartState {
  switch (action.type) {
    case "hydrate":
      return action.items;
    case "add": {
      const existing = state.find(
        (i) => i.productVariantId === action.item.productVariantId
      );
      if (existing) {
        return state.map((i) =>
          i.productVariantId === action.item.productVariantId
            ? {
                ...i,
                quantity: Math.min(
                  i.quantity + action.item.quantity,
                  i.stockQuantity
                ),
              }
            : i
        );
      }
      return [...state, action.item];
    }
    case "updateQuantity":
      return state.map((i) =>
        i.productVariantId === action.productVariantId
          ? {
              ...i,
              quantity: Math.max(
                1,
                Math.min(action.quantity, i.stockQuantity)
              ),
            }
          : i
      );
    case "remove":
      return state.filter((i) => i.productVariantId !== action.productVariantId);
    case "clear":
      return [];
    default:
      return state;
  }
}

type CartContextValue = {
  items: CartItem[];
  itemCount: number;
  subtotal: number;
  addItem: (item: CartItem) => void;
  updateQuantity: (productVariantId: string, quantity: number) => void;
  removeItem: (productVariantId: string) => void;
  clear: () => void;
};

const CartContext = createContext<CartContextValue | null>(null);

export function CartProvider({ children }: { children: React.ReactNode }) {
  const [items, dispatch] = useReducer(cartReducer, []);

  // Cart is client-only state; hydrate from localStorage after mount to
  // avoid a server/client render mismatch.
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      if (raw) {
        dispatch({ type: "hydrate", items: JSON.parse(raw) as CartItem[] });
      }
    } catch {
      // ignore malformed/unavailable localStorage
    }
  }, []);

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    } catch {
      // ignore write failures (e.g. private browsing quota)
    }
  }, [items]);

  const value = useMemo<CartContextValue>(() => {
    return {
      items,
      itemCount: items.reduce((sum, i) => sum + i.quantity, 0),
      subtotal: items.reduce((sum, i) => sum + i.price * i.quantity, 0),
      addItem: (item) => dispatch({ type: "add", item }),
      updateQuantity: (productVariantId, quantity) =>
        dispatch({ type: "updateQuantity", productVariantId, quantity }),
      removeItem: (productVariantId) =>
        dispatch({ type: "remove", productVariantId }),
      clear: () => dispatch({ type: "clear" }),
    };
  }, [items]);

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) {
    throw new Error("useCart must be used within a CartProvider");
  }
  return ctx;
}
