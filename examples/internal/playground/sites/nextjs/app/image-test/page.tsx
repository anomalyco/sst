import Image from "next/image";
import Link from "next/link";
import styles from "../page.module.css";

export default function ImageTest() {
  return (
    <div className={styles.page} style={{ backgroundColor: "white" }}>
      <main className={styles.main}>
        <h1>Next.js Image Optimization Test</h1>
        <div style={{ marginBottom: "20px" }}>
          <Link href="/" style={{ textDecoration: "underline" }}>
            Back to Home
          </Link>
        </div>

        <h2>Image from public folder:</h2>
        <div style={{ position: "relative", width: "300px", height: "300px" }}>
          <Image
            src="/black.jpg"
            alt="Black square"
            width={300}
            height={300}
            style={{ objectFit: "contain" }}
            priority
          />
        </div>

        <h2>Remote image (will use optimization):</h2>
        <div
          style={{
            marginTop: "30px",
            position: "relative",
            width: "500px",
            height: "300px",
          }}
        >
          <Image
            src="https://images.unsplash.com/photo-1579546929518-9e396f3cc809"
            alt="Remote image"
            width={500}
            height={300}
            priority
          />
        </div>
      </main>
    </div>
  );
}
