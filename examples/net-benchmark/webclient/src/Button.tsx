import React from "react";

interface ButtonProps {
  onClick?: () => void;
  children: React.ReactNode;
  disabled?: boolean;
}

export const Button: React.FC<ButtonProps> = React.memo(
  ({ onClick, children, disabled }) => {
    const [hover, setHover] = React.useState(false);

    return (
      <div
        style={{
          marginLeft: "auto",
          backgroundColor: "#4538e5",
          padding: "8px 28px",
          border: "1px solid #eee",
          borderRadius: "8px",
          fontWeight: "700",
          fontSize: "14pt",
          cursor: "pointer",
          userSelect: "none",
          ...(hover && {
            backgroundColor: "#1C09EE",
            border: "1px solid #1C09EE",
            color: "#fff",
          }),
          ...(disabled && {
            backgroundColor: "#333",
            color: "#aaa",
            cursor: "not-allowed",
          }),
        }}
        onMouseEnter={() => setHover(true)}
        onMouseLeave={() => setHover(false)}
        onClick={() => !disabled && onClick?.()}
      >
        {children}
      </div>
    );
  }
);
