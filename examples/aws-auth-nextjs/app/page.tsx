import Image from "next/image";
import styles from "./page.module.css";
import { auth, login, logout } from "./actions";

export default async function Home() {
  const subject = await auth();

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <Image
          className={styles.logo}
          src="/next.svg"
          alt="Next.js logo"
          width={180}
          height={38}
          priority
        />
        <ol>
          {subject ? (
            <>
              <li>
                Logged in as <code>{subject.properties.id}</code>.
              </li>
              <li>
                And then check out <code>app/page.tsx</code>.
              </li>
            </>
          ) : (
            <>
              <li>Login with your email and password.</li>
              <li>
                And then check out <code>app/page.tsx</code>.
              </li>
            </>
          )}
        </ol>

        <div className={styles.ctas}>
          {subject ? (
            <form action={logout}>
              <button className={styles.secondary}>Logout</button>
            </form>
          ) : (
            <form action={login}>
              <button className={styles.primary}>Login with OpenAuth</button>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}
